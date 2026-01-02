
from __future__ import annotations
import argparse, logging, sys, shutil
from pathlib import Path
from datetime import datetime
import pandas as pd
from scripts.pass1_parse_json import run_pass1
from scripts.pass2_sort_records import assign_ids
from scripts.pass3_analytics import compute_analytics
from scripts.pass4_link_points import link_points
from scripts.pass4_5_generate_api_orders import generate_orders

PROJECT_ROOT = Path(__file__).resolve().parents[1]
DEFAULT_SCHEMA = PROJECT_ROOT / "schemas/google_2024.json"
LOGS_DIR = PROJECT_ROOT / "logs"

REQUIRED_DIRS = [
    "source_data",
    "raw_api_responses",
    "output_data/master_csvs/default",
    "output_data/api_orders/default",
    "output_data/human_readable_csvs/default",
    "working_data/pass1_flatten_json",
    "working_data/pass2_sorted_records",
    "working_data/pass3_analytics",
    "working_data/pass4_linked_points",
    "schemas",
    "logs",
]

def ensure_dirs():
    for d in REQUIRED_DIRS:
        (PROJECT_ROOT / d).mkdir(parents=True, exist_ok=True)

def discover_sources(src_arg: str|None):
    sd = PROJECT_ROOT / "source_data"
    if src_arg:
        p = Path(src_arg)
        if p.is_dir():
            return sorted(list(p.glob("*.json")))
        return [p]
    return sorted(list(sd.glob("*.json")))

def auto_out_basename(src_path: Path) -> str:
    return src_path.stem

def preview_csv(df: pd.DataFrame, path: Path):
    df.head(25).to_csv(path, index=False)

def main():
    ensure_dirs()
    LOGS_DIR.mkdir(parents=True, exist_ok=True)
    logging.basicConfig(
        level=logging.INFO,
        handlers=[
            logging.FileHandler(LOGS_DIR / f"orchestrator_{datetime.now().date()}.log"),
            logging.StreamHandler(sys.stdout),
        ],
        format="%(asctime)s [%(levelname)s] %(message)s"
    )
    ap = argparse.ArgumentParser()
    ap.add_argument("--src", help="source file or folder; defaults to source_data/")
    ap.add_argument("--schema", default=str(DEFAULT_SCHEMA))
    ap.add_argument("--tag", default="default")
    ap.add_argument("--no-api-orders", action="store_true", help="Skip Pass 4.5 API order generation")
    args = ap.parse_args()

    schema_path = Path(args.schema)
    if not schema_path.exists():
        logging.error("Schema not found: %s", schema_path)
        sys.exit(2)

    sources = discover_sources(args.src)
    if not sources:
        logging.warning("No source JSON found. Drop files into source_data/ and rerun.")
        sys.exit(0)

    for src in sources:
        logging.info("Processing %s", src)
        base = auto_out_basename(src)

        # Pass 1
        df1 = run_pass1(src, schema_path)
        p1 = PROJECT_ROOT / f"working_data/pass1_flatten_json/{base}.csv"
        df1.to_csv(p1, index=False)
        preview_csv(df1, p1.with_name(f"{base}.preview.csv"))

        # Pass 2
        df2 = assign_ids(df1)
        p2 = PROJECT_ROOT / f"working_data/pass2_sorted_records/{base}.csv"
        df2.to_csv(p2, index=False)
        preview_csv(df2, p2.with_name(f"{base}.preview.csv"))

        # Pass 3 (analytics) - BEFORE linking
        df3 = compute_analytics(df2)
        p3 = PROJECT_ROOT / f"working_data/pass3_analytics/{base}.csv"
        df3.to_csv(p3, index=False)
        preview_csv(df3, p3.with_name(f"{base}.preview.csv"))

        # Pass 4 - link points to parent paths
        df4 = link_points(df3)
        p4 = PROJECT_ROOT / f"working_data/pass4_linked_points/{base}.csv"
        df4.to_csv(p4, index=False)
        preview_csv(df4, p4.with_name(f"{base}.preview.csv"))

        # Pass 4.5 - API orders (optional)
        if not args.no-api_orders:
            orders_dir = PROJECT_ROOT / f"output_data/api_orders/{args.tag}/{base}"
            generate_orders(df4, orders_dir)

        # Human-readable CSV = Pass 4 (linked, post-analytics)
        h = PROJECT_ROOT / f"output_data/human_readable_csvs/{args.tag}/{base}.csv"
        df4.to_csv(h, index=False)

        # Move processed source to completed/
        completed_dir = PROJECT_ROOT / "source_data/completed"
        completed_dir.mkdir(parents=True, exist_ok=True)
        shutil.copy2(src, completed_dir / src.name)
        logging.info("Done %s -> moved copy to %s", src.name, completed_dir)

    logging.info("All done.")

if __name__ == "__main__":
    main()
