from __future__ import annotations
import sys, json
from pathlib import Path
from datetime import datetime
from typing import List, Dict, Any, Optional

import pandas as pd
from PySide6.QtCore import Qt, QDate, QThread, Signal, QFileSystemWatcher
from PySide6.QtWidgets import (
    QApplication, QMainWindow, QWidget, QFileDialog, QVBoxLayout, QHBoxLayout,
    QPushButton, QLabel, QTableWidget, QTableWidgetItem, QTextEdit, QListWidget,
    QCheckBox, QDateEdit, QSpinBox, QMessageBox, QLineEdit, QSplitter
)

from parsers import parse_sms_xml, parse_html_messages, parse_pdf_messages, stream_sms_xml
from rules_loader import load_rules_from_dir
from taggers import RuleEngine
from sequences import detect_sequences

class ParseWorker(QThread):
    finished = Signal(pd.DataFrame, dict)
    failed = Signal(str)
    def __init__(self, files: List[Path], recursive: bool, since: datetime, until: datetime,
                 page_limit: Optional[int], do_tag: bool, engine: RuleEngine):
        super().__init__()
        self.files = files
        self.recursive = recursive
        self.since = since
        self.until = until
        self.page_limit = page_limit
        self.do_tag = do_tag
        self.engine = engine

    @staticmethod
    def discover_files(entries: List[Path], recursive: bool) -> List[Path]:
        out: List[Path] = []
        for e in entries:
            if e.is_dir():
                glob = "**/*" if recursive else "*"
                for p in e.glob(glob):
                    if p.suffix.lower() in {'.pdf', '.html', '.htm', '.xml'}:
                        out.append(p)
            elif e.is_file() and e.suffix.lower() in {'.pdf', '.html', '.htm', '.xml'}:
                out.append(e)
        # Unique + sorted
        return sorted({p.resolve() for p in out})

    def run(self):
        try:

            files = self.discover_files(self.files, self.recursive)
            # streaming mode writes to temp list with cap; full set in memory otherwise
            records = []
            cap = 100000
            try:
                cap = int(getattr(self, 'cap_rows', cap))
            except Exception:
                pass
            total = 0
            for p in files:
                ext = p.suffix.lower()
                try:
                    if ext == '.xml':
                        # Stream parse
                        for row in stream_sms_xml(p, p.name):
                            dt = row.get('datetime')
                            if dt is None or dt < self.since or dt > self.until: 
                                continue
                            total += 1
                            if len(records) < cap:
                                records.append(row)
                    elif ext in ('.html', '.htm'):
                        recs = parse_html_messages(p, p.name)
                        for r in recs:
                            dt = r.get('datetime')
                            if dt is None or dt < self.since or dt > self.until: continue
                            total += 1
                            if len(records) < cap: records.append(r)
                    elif ext == '.pdf':
                        recs = parse_pdf_messages(p, p.name, page_limit=self.page_limit)
                        for r in recs:
                            dt = r.get('datetime')
                            if dt is None or dt < self.since or dt > self.until: continue
                            total += 1
                            if len(records) < cap: records.append(r)
                except Exception:
                    continue
            if not records:
                self.failed.emit('No messages parsed from selected sources with current filters.')
                return
            df = pd.DataFrame(records)
            df = df.drop_duplicates(subset=['datetime','sender','message']).sort_values('datetime').reset_index(drop=True)
            # normalize sender names to roles where possible
            df['sender_role'] = df['sender'].apply(lambda s: self.engine.normalize_sender(str(s)))
            if self.do_tag:
                df['behavior_tags'] = df['message'].apply(lambda t: ",".join(self.engine.tag_behaviors(str(t))))
                df['mcl_factors'] = df['message'].apply(lambda t: ",".join(self.engine.tag_mcl(str(t))))
                df['sentiment'] = df['message'].apply(lambda t: self.engine.sentiment(str(t)))
            summary = (df.groupby('source').agg(first=('datetime','min'), last=('datetime','max'), count=('datetime','count'))
                        .reset_index().sort_values('first'))
            self.finished.emit(df, summary.to_dict(orient='records'))
        except Exception as e:
            self.failed.emit(str(e))

class MainWindow(QMainWindow):
    def __init__(self):
        super().__init__()
        self.setWindowTitle('Conflict Analysis App')
        self.resize(1400, 900)

        self.files: List[Path] = []
        self.rules_dir: Optional[Path] = Path.cwd() / 'rules' if (Path.cwd() / 'rules').exists() else None
        self.rules = load_rules_from_dir(self.rules_dir) if self.rules_dir else {'behaviors':{},'mcl':{}}
        self.engine = RuleEngine(self.rules)

        # File watcher for hot reload
        self.watcher = QFileSystemWatcher(self)
        if self.rules_dir:
            self._watch_rules_dir(self.rules_dir)

        # Controls
        self.btn_add_files = QPushButton('Add Files…')
        self.btn_add_folder = QPushButton('Add Folder…')
        self.chk_recursive = QCheckBox('Recursive'); self.chk_recursive.setChecked(True)
        self.btn_clear = QPushButton('Clear List')

        self.since = QDateEdit(); self.since.setDisplayFormat('yyyy-MM-dd'); self.since.setCalendarPopup(True); self.since.setDate(QDate(2018,1,1))
        self.until = QDateEdit(); self.until.setDisplayFormat('yyyy-MM-dd'); self.until.setCalendarPopup(True); self.until.setDate(QDate.currentDate())
        self.spin_pages = QSpinBox(); self.spin_pages.setRange(0, 2000); self.spin_pages.setValue(80)
        self.lbl_pages = QLabel('PDF pages (0=all):')

        self.chk_tag = QCheckBox('Enable tagging + sentiment')
        self.chk_stream = QCheckBox('Stream (low memory)')
        self.mem_cap = QSpinBox(); self.mem_cap.setRange(1000, 2000000); self.mem_cap.setValue(100000)
        self.lbl_cap = QLabel('Max rows in memory:')
        self.btn_view_parties = QPushButton('View Parties/Notes')

        self.rules_dir_edit = QLineEdit(str(self.rules_dir) if self.rules_dir else ''); self.rules_dir_edit.setPlaceholderText('rules directory')
        self.btn_rules = QPushButton('Select Rules Folder…')

        self.btn_scan = QPushButton('Scan')
        self.btn_sequences = QPushButton('Detect Sequences')
        self.btn_export = QPushButton('Export…')

        # Lists/Tables
        self.list_files = QListWidget()
        self.table = QTableWidget(0, 7); self.table.setHorizontalHeaderLabels(['#','datetime','sender','role','message','source','behavior_tags'])
        self.table.setSortingEnabled(True)
        self.seqs_table = QTableWidget(0, 6); self.seqs_table.setHorizontalHeaderLabels(['type','score','start','end','participants','#msgs'])
        self.preview = QTextEdit(); self.preview.setReadOnly(True)

        # Layout
        top = QHBoxLayout()
        top.addWidget(self.btn_add_files); top.addWidget(self.btn_add_folder); top.addWidget(self.chk_recursive); top.addWidget(self.btn_clear)

        filters = QHBoxLayout()
        filters.addWidget(QLabel('Since')); filters.addWidget(self.since)
        filters.addWidget(QLabel('Until')); filters.addWidget(self.until)
        filters.addWidget(self.lbl_pages); filters.addWidget(self.spin_pages)

        rules = QHBoxLayout()
        rules.addWidget(self.chk_tag); rules.addWidget(self.chk_stream); rules.addWidget(self.lbl_cap); rules.addWidget(self.mem_cap); rules.addWidget(self.rules_dir_edit); rules.addWidget(self.btn_rules); rules.addWidget(self.btn_view_parties); rules.addStretch()

        actions = QHBoxLayout()
        actions.addWidget(self.btn_scan); actions.addWidget(self.btn_sequences); actions.addWidget(self.btn_export); actions.addStretch()

        left = QVBoxLayout()
        left.addLayout(top); left.addLayout(filters); left.addLayout(rules); left.addLayout(actions)
        left.addWidget(QLabel('Files to scan')); left.addWidget(self.list_files, 1)

        right_top = QVBoxLayout()
        right_top.addWidget(QLabel('Parsed messages'))
        right_top.addWidget(self.table, 3)
        right_bottom = QVBoxLayout()
        right_bottom.addWidget(QLabel('Detected sequences'))
        right_bottom.addWidget(self.seqs_table, 2)
        right_bottom.addWidget(QLabel('Preview'))
        right_bottom.addWidget(self.preview, 2)

        right_container = QWidget(); right_layout = QVBoxLayout(right_container)
        right_layout.addLayout(right_top)
        right_layout.addLayout(right_bottom)

        root = QHBoxLayout()
        root.addLayout(left, 1)
        root.addWidget(right_container, 2)
        container = QWidget(); container.setLayout(root)
        self.setCentralWidget(container)

        # Events
        self.btn_add_files.clicked.connect(self.on_add_files)
        self.btn_add_folder.clicked.connect(self.on_add_folder)
        self.btn_clear.clicked.connect(self.on_clear)
        self.btn_rules.clicked.connect(self.on_load_rules)
        self.btn_scan.clicked.connect(self.on_scan)
        self.btn_sequences.clicked.connect(self.on_sequences)
        self.btn_export.clicked.connect(self.on_export)
        self.btn_view_parties.clicked.connect(self.on_view_parties)
        self.table.itemSelectionChanged.connect(self.on_row_selected)

        self.df: Optional[pd.DataFrame] = None
        self.summary_rows: List[Dict[str, Any]] = []

    def _watch_rules_dir(self, d: Path):
        # clear existing
        self.watcher.removePaths(self.watcher.files())
        self.watcher.removePaths(self.watcher.directories())
        self.watcher.addPath(str(d))
        for p in d.glob("*.yaml"):
            self.watcher.addPath(str(p))
        self.watcher.directoryChanged.connect(self._reload_rules)
        self.watcher.fileChanged.connect(self._reload_rules)

    def _reload_rules(self, *_):
        if not self.rules_dir: return
        try:
            self.rules = load_rules_from_dir(self.rules_dir)
            self.engine = RuleEngine(self.rules)
            QMessageBox.information(self, 'Rules reloaded', f'Hot reloaded rules from\n{self.rules_dir}')
            # Re-apply tags if data is present
            if self.df is not None and not self.df.empty and self.chk_tag.isChecked():
                self.df['behavior_tags'] = self.df['message'].apply(lambda t: ",".join(self.engine.tag_behaviors(str(t))))
                self.df['mcl_factors'] = self.df['message'].apply(lambda t: ",".join(self.engine.tag_mcl(str(t))))
                self.df['sentiment'] = self.df['message'].apply(lambda t: self.engine.sentiment(str(t)))
                self._populate_table(self.df)
        except Exception as e:
            QMessageBox.critical(self, 'Reload failed', str(e))

    # ---- UI Handlers ----
    def on_add_files(self):
        files, _ = QFileDialog.getOpenFileNames(self, 'Select message exports', str(Path.home()),
                    'Exports (*.pdf *.html *.htm *.xml);;All Files (*)')
        for f in files:
            p = Path(f)
            if p.exists():
                self.files.append(p)
                self.list_files.addItem(str(p))

    def on_add_folder(self):
        folder = QFileDialog.getExistingDirectory(self, 'Select folder to scan', str(Path.home()))
        if folder:
            p = Path(folder)
            self.files.append(p)
            self.list_files.addItem(str(p))

    def on_clear(self):
        self.files = []
        self.list_files.clear()
        self.table.setRowCount(0)
        self.seqs_table.setRowCount(0)
        self.preview.clear()
        self.df = None

    def on_load_rules(self):
        d = QFileDialog.getExistingDirectory(self, 'Select rules folder', str(self.rules_dir or Path.cwd()))
        if not d: return
        self.rules_dir = Path(d)
        self.rules_dir_edit.setText(str(self.rules_dir))
        self.rules = load_rules_from_dir(self.rules_dir)
        self.engine = RuleEngine(self.rules)
        self._watch_rules_dir(self.rules_dir)
        QMessageBox.information(self, 'Rules loaded', f'Loaded rules from\n{d}')

    def on_scan(self):
        if not self.files:
            QMessageBox.warning(self, 'No files', 'Add files or a folder to scan.')
            return
        since_dt = datetime(self.since.date().year(), self.since.date().month(), self.since.date().day())
        until_dt = datetime(self.until.date().year(), self.until.date().month(), self.until.date().day(), 23,59,59)
        page_limit = self.spin_pages.value()
        if page_limit == 0: page_limit = None
        self.worker = ParseWorker(self.files, self.chk_recursive.isChecked(), since_dt, until_dt,
                                  page_limit, self.chk_tag.isChecked(), self.engine)
        # pass cap from UI
        self.worker.cap_rows = self.mem_cap.value()
        self.worker.finished.connect(self.on_parsed)
        self.worker.failed.connect(self.on_failed)
        self.btn_scan.setEnabled(False)
        self.worker.start()

    def on_parsed(self, df: pd.DataFrame, summary_rows):
        self.btn_scan.setEnabled(True)
        self.df = df
        self.summary_rows = summary_rows
        self._populate_table(df)

    def _populate_table(self, df: pd.DataFrame):
        self.table.setRowCount(len(df))
        for r, row in df.reset_index().iterrows():
            self.table.setItem(r, 0, QTableWidgetItem(str(row['index'])))
            self.table.setItem(r, 1, QTableWidgetItem(str(row['datetime'])))
            self.table.setItem(r, 2, QTableWidgetItem(str(row.get('sender',''))))
            self.table.setItem(r, 3, QTableWidgetItem(str(row.get('sender_role',''))))
            self.table.setItem(r, 4, QTableWidgetItem(str(row.get('message',''))))
            self.table.setItem(r, 5, QTableWidgetItem(str(row.get('source',''))))
            self.table.setItem(r, 6, QTableWidgetItem(str(row.get('behavior_tags',''))))

    def on_failed(self, msg: str):
        self.btn_scan.setEnabled(True)
        QMessageBox.critical(self, 'Parse failed', msg)

    def on_row_selected(self):
        sel = self.table.selectedItems()
        if not sel:
            self.preview.clear()
            return
        r = sel[0].row()
        msg = self.table.item(r, 3).text() if self.table.item(r,3) else ''
        self.preview.setPlainText(msg)

    def on_view_parties(self):
        roles = (self.rules.get('parties') or {}).get('roles') or {}
        if not roles:
            QMessageBox.information(self, 'Parties', 'No parties.yaml loaded.'); return
        lines = []
        for role, obj in roles.items():
            names = ', '.join(obj.get('names') or [])
            notes = obj.get('notes','')
            lines.append(f"{role}: {names}\n  {notes}")
        QMessageBox.information(self, 'Parties', "\n\n".join(lines))

    def on_sequences(self):
        if self.df is None or self.df.empty:
            QMessageBox.warning(self, 'No data', 'Run a scan first.')
            return
        seqs_rules = load_rules_from_dir(self.rules_dir).get('sequences', {}) if self.rules_dir else {}
        events = detect_sequences(self.df, seqs_rules)
        self.seqs_table.setRowCount(len(events))
        for i, ev in enumerate(events):
            self.seqs_table.setItem(i, 0, QTableWidgetItem(ev['sequence_type']))
            self.seqs_table.setItem(i, 1, QTableWidgetItem(str(ev.get('score',1))))
            self.seqs_table.setItem(i, 2, QTableWidgetItem(ev['start_datetime']))
            self.seqs_table.setItem(i, 3, QTableWidgetItem(ev['end_datetime']))
            self.seqs_table.setItem(i, 4, QTableWidgetItem(", ".join(ev.get('participants',[]))))
            self.seqs_table.setItem(i, 5, QTableWidgetItem(str(len(ev.get('message_indices',[])))))
        # stash for export
        self.seq_events = events

    def on_export(self):
        if self.df is None or self.df.empty:
            QMessageBox.warning(self, 'Nothing to export', 'Run a scan first.')
            return
        out_dir = QFileDialog.getExistingDirectory(self, 'Select export folder', str(Path.cwd()))
        if not out_dir: return
        out = Path(out_dir)
        # Master
        master_path = out / 'master_messages.csv'
        self.df[['datetime','sender','message','source']].to_csv(master_path, index=False)
        # Tagged & signal
        if 'behavior_tags' in self.df.columns or 'mcl_factors' in self.df.columns:
            tagged_path = out / 'tagged_messages.csv'
            self.df.to_csv(tagged_path, index=False)
            signal = self.df[(self.df.get('behavior_tags','')!='') | (self.df.get('mcl_factors','')!='')].copy()
            (out / 'signal_messages.csv').write_text(signal.to_csv(index=False), encoding='utf-8')
        # Sequences
        seqs_json = out / 'sequences.json'
        events = getattr(self, 'seq_events', [])
        seqs_json.write_text(json.dumps(events, indent=2), encoding='utf-8')
        QMessageBox.information(self, 'Export complete', f'Wrote files to:\n{out}')

if __name__ == '__main__':
    app = QApplication(sys.argv)
    win = MainWindow()
    win.show()
    sys.exit(app.exec())
