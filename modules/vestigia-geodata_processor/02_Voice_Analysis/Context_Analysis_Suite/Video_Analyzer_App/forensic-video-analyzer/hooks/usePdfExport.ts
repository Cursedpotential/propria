
import { useState, useCallback } from 'react';
import jsPDF from 'jspdf';
import html2canvas from 'html2canvas';

export const usePdfExport = (elementId: string, fileNamePrefix: string) => {
  const [isExporting, setIsExporting] = useState(false);

  const exportToPdf = useCallback(async () => {
    const input = document.getElementById(elementId);
    if (!input) {
      console.error("Element not found for PDF export:", elementId);
      return;
    }

    setIsExporting(true);

    try {
      const canvas = await html2canvas(input, {
        scale: 2, // Higher scale for better quality
        useCORS: true,
        backgroundColor: '#1e293b', // Match the dark background
      });

      const imgData = canvas.toDataURL('image/png');
      const pdf = new jsPDF({
        orientation: 'p',
        unit: 'px',
        format: [canvas.width, canvas.height]
      });

      pdf.addImage(imgData, 'PNG', 0, 0, canvas.width, canvas.height);
      const fileName = `${fileNamePrefix.replace(/[^a-z0-9]/gi, '_').toLowerCase()}.pdf`;
      pdf.save(fileName);

    } catch (error) {
      console.error("Error exporting to PDF:", error);
    } finally {
      setIsExporting(false);
    }
  }, [elementId, fileNamePrefix]);

  return { exportToPdf, isExporting };
};
