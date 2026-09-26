import React, { useEffect, useState } from 'react';
import { supabase } from './supabaseClient';
import { 
  DataGrid, 
  GridColDef, 
  GridRenderCellParams, 
  GridToolbarContainer, 
  GridToolbarFilterButton, 
  GridToolbarQuickFilter 
} from '@mui/x-data-grid';
import { 
  Container, Typography, Chip, Dialog, DialogTitle, DialogContent, 
  TextField, DialogActions, Button, Snackbar, Alert, Box, 
  CircularProgress, IconButton, Tooltip, MenuItem 
} from '@mui/material';
import EditIcon from '@mui/icons-material/Edit';
import AutoAwesomeIcon from '@mui/icons-material/AutoAwesome';
import DownloadIcon from '@mui/icons-material/Download';
import LinkIcon from '@mui/icons-material/Link';
import TerminalIcon from '@mui/icons-material/Terminal';
import { GoogleGenerativeAI } from "@google/generative-ai";

const genAI = new GoogleGenerativeAI(import.meta.env.VITE_GEMINI_API_KEY);

interface LocationData {
  id: string;
  address: string;
  lat: number;
  lng: number;
  risk_level: string | null;
  risk_notes: string | null;
  place_id: string;
  types?: any;
  linked_entity_id: string | null;
  entity_name: string | null;
}

// --- Custom Toolbar with Export ---
function CustomToolbar({ rows }: { rows: LocationData[] }) {
  const handleExport = () => {
    const headers = ['Address', 'Entity Name', 'Risk Level', 'Notes', 'Categories', 'Lat', 'Lng', 'Entity ID', 'Place ID'];
    const csvContent = [
      headers.join(','),
      ...rows.map(row => [
        `"${(row.address || '').replace(/"/g, '""')}"`,
        `"${(row.entity_name || '').replace(/"/g, '""')}"`,
        `"${row.risk_level || ''}"`,
        `"${(row.risk_notes || '').replace(/"/g, '""')}"`,
        `"${Array.isArray(row.types) ? row.types.join(', ') : (row.types || '')}"`,
        row.lat,
        row.lng,
        row.linked_entity_id || '',
        row.place_id
      ].join(','))
    ].join('\n');

    const blob = new Blob([csvContent], { type: 'text/csv;charset=utf-8;' });
    const link = document.createElement('a');
    link.href = URL.createObjectURL(blob);
    link.download = `locations_export_${new Date().toISOString().slice(0,10)}.csv`;
    link.click();
  };

  return (
    <GridToolbarContainer sx={{ p: 1, gap: 1, borderBottom: '1px solid #333' }}>
      <GridToolbarQuickFilter />
      <GridToolbarFilterButton />
      <Button startIcon={<DownloadIcon />} size="small" onClick={handleExport}>
        Export CSV
      </Button>
    </GridToolbarContainer>
  );
}

function App() {
  const [rows, setRows] = useState<LocationData[]>([]);
  const [open, setOpen] = useState(false);
  const [currentRow, setCurrentRow] = useState<LocationData | null>(null);
  const [formData, setFormData] = useState({ 
    risk_level: '', 
    risk_notes: '', 
    entity_name: '', 
    linked_entity_id: '' 
  });
  const [toast, setToast] = useState({ open: false, message: '', severity: 'success' as any });
  const [analyzing, setAnalyzing] = useState(false);
  const [selectedAgent, setSelectedAgent] = useState('direct-gemini');

  useEffect(() => {
    fetchLocations();
  }, []);

  const fetchLocations = async () => {
    const { data, error } = await supabase
      .from('geocode_cache_google')
      .select('address, latitude, longitude, risk_level, risk_notes, place_id, types, linked_entity_id, entity_name')
      .limit(2000);

    if (error) {
      setToast({ open: true, message: 'Fetch Error: ' + error.message, severity: 'error' });
    } else {
      const mappedData = data.map((row: any, index: number) => ({
        id: row.place_id || row.address || index.toString(),
        address: row.address,
        lat: row.latitude,
        lng: row.longitude,
        risk_level: row.risk_level,
        risk_notes: row.risk_notes,
        place_id: row.place_id,
        types: row.types, 
        linked_entity_id: row.linked_entity_id,
        entity_name: row.entity_name
      }));
      setRows(mappedData);
    }
  };

  const handleEditClick = (row: LocationData) => {
    setCurrentRow(row);
    setFormData({ 
      risk_level: row.risk_level || '', 
      risk_notes: row.risk_notes || '',
      entity_name: row.entity_name || '',
      linked_entity_id: row.linked_entity_id || ''
    });
    setOpen(true);
  };

  const handleAnalysis = async () => {
    if (!currentRow) return;
    setAnalyzing(true);
    
    const prompt = `Return JSON only: { "risk_level": "LOW|MEDIUM|HIGH|RESTRICTED", "notes": "brief notes", "entity_name": "named place" }
    Analyze: ${currentRow.address} | Types: ${JSON.stringify(currentRow.types)}`;

    try {
      let resultText = '';
      if (selectedAgent === 'direct-gemini') {
        const model = genAI.getGenerativeModel({ model: "gemini-1.5-flash" });
        const result = await model.generateContent(prompt);
        resultText = result.response.text();
      } else {
        const response = await fetch('http://localhost:3001/api/ask-agent', {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({ agent: selectedAgent, prompt })
        });
        const json = await response.json();
        if (!json.success) throw new Error(json.error);
        resultText = json.data;
      }

      const match = resultText.match(/\{[\s\S]*\}/);
      if (!match) throw new Error("No JSON found in response");
      const analysis = JSON.parse(match[0]);

      setFormData(prev => ({
        ...prev,
        risk_level: analysis.risk_level || prev.risk_level,
        risk_notes: analysis.notes || prev.risk_notes,
        entity_name: analysis.entity_name || prev.entity_name
      }));
      setToast({ open: true, message: `Analysis by ${selectedAgent} applied`, severity: 'info' });
    } catch (error: any) {
      setToast({ open: true, message: 'Analysis Failed: ' + error.message, severity: 'error' });
    } finally {
      setAnalyzing(false);
    }
  };

  const handleSave = async () => {
    if (!currentRow) return;
    const { error } = await supabase
      .from('geocode_cache_google')
      .update({
        risk_level: formData.risk_level || null, 
        risk_notes: formData.risk_notes || null,
        entity_name: formData.entity_name || null,
        linked_entity_id: formData.linked_entity_id || null
      })
      .eq('place_id', currentRow.place_id);

    if (error) {
      setToast({ open: true, message: 'Save Error: ' + error.message, severity: 'error' });
    } else {
      setToast({ open: true, message: 'Saved successfully', severity: 'success' });
      setRows(rows.map(r => r.id === currentRow.id ? { ...r, ...formData } : r));
      setOpen(false);
    }
  };

  const columns: GridColDef[] = [
    { field: 'address', headerName: 'Address', flex: 2 },
    { field: 'entity_name', headerName: 'Entity / Name', flex: 1 },
    {
      field: 'types', 
      headerName: 'Categories', 
      flex: 1,
      valueGetter: (params) => Array.isArray(params.row.types) ? params.row.types.join(', ') : params.row.types
    },
    {
      field: 'risk_level', 
      headerName: 'Risk', 
      width: 120,
      renderCell: (params) => params.value ? <Chip label={params.value} size="small" /> : null
    },
    { field: 'risk_notes', headerName: 'Notes', flex: 1.5 },
    {
      field: 'actions',
      headerName: 'Actions',
      width: 80,
      renderCell: (params) => (
        <IconButton onClick={() => handleEditClick(params.row as LocationData)} color="primary" size="small">
          <EditIcon />
        </IconButton>
      ),
    },
  ];

  return (
    <Container maxWidth="xl" sx={{ height: '100vh', py: 4, display: 'flex', flexDirection: 'column' }}>
      <Typography variant="h5" gutterBottom sx={{ fontWeight: 'bold', color: '#90caf9', mb: 2 }}>
        TraceIQ Location Manager
      </Typography>
      <Box sx={{ flex: 1, width: '100%', bgcolor: '#1e1e1e', borderRadius: 2, overflow: 'hidden' }}>
        <DataGrid
          rows={rows}
          columns={columns}
          slots={{ toolbar: CustomToolbar }}
          slotProps={{ toolbar: { rows } }}
          disableRowSelectionOnClick
          sx={{ border: 0, color: 'text.secondary' }}
        />
      </Box>

      <Dialog open={open} onClose={() => setOpen(false)} maxWidth="md" fullWidth>
        <DialogTitle sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
          Edit Location Details
          <Box sx={{ display: 'flex', gap: 1 }}>
            <TextField select size="small" value={selectedAgent} onChange={(e) => setSelectedAgent(e.target.value)} sx={{ minWidth: 150 }}>
              <MenuItem value="direct-gemini">Gemini (Direct)</MenuItem>
              <MenuItem value="gemini-cli">Gemini (CLI)</MenuItem>
              <MenuItem value="claude">Claude Code</MenuItem>
              <MenuItem value="qwen">Qwen CLI</MenuItem>
            </TextField>
            <Button variant="outlined" color="secondary" startIcon={analyzing ? <CircularProgress size={20} /> : <TerminalIcon />} onClick={handleAnalysis} disabled={analyzing}>
               Run
            </Button>
          </Box>
        </DialogTitle>
        <DialogContent dividers>
          <Box sx={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 3 }}>
            <Box sx={{ display: 'flex', flexDirection: 'column', gap: 2 }}>
              <Typography variant="body2">{currentRow?.address}</Typography>
              <TextField label="Entity Name" fullWidth value={formData.entity_name} onChange={(e) => setFormData({ ...formData, entity_name: e.target.value })} />
              <TextField label="Linked Profile UUID" fullWidth value={formData.linked_entity_id} onChange={(e) => setFormData({ ...formData, linked_entity_id: e.target.value })} />
            </Box>
            <Box sx={{ display: 'flex', flexDirection: 'column', gap: 2 }}>
              <TextField label="Risk Level" select SelectProps={{ native: true }} fullWidth value={formData.risk_level} onChange={(e) => setFormData({ ...formData, risk_level: e.target.value }) }>
                <option value="">Unset</option>
                <option value="LOW">LOW</option>
                <option value="MEDIUM">MEDIUM</option>
                <option value="HIGH">HIGH</option>
                <option value="RESTRICTED">RESTRICTED</option>
              </TextField>
              <TextField label="Notes" fullWidth multiline rows={6} value={formData.risk_notes} onChange={(e) => setFormData({ ...formData, risk_notes: e.target.value })} />
            </Box>
          </Box>
        </DialogContent>
        <DialogActions>
          <Button onClick={() => setOpen(false)}>Cancel</Button>
          <Button onClick={handleSave} variant="contained">Save</Button>
        </DialogActions>
      </Dialog>
      <Snackbar open={toast.open} autoHideDuration={6000} onClose={() => setToast({ ...toast, open: false })}>
        <Alert severity={toast.severity}>{toast.message}</Alert>
      </Snackbar>
    </Container>
  );
}

export default App;
