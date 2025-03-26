// No admin rights needed—include ExcelJS via a CDN in your HTML file
// Example: in your HTML <head>, add:
// <script src="https://unpkg.com/exceljs/dist/exceljs.min.js"></script>

async function exportDetailedDaily(tableData) {
  const workbook = new ExcelJS.Workbook();
  const worksheet = workbook.addWorksheet('Detailed Daily');

  // Write data into the worksheet and apply a thin black border to every cell
  tableData.forEach(row => {
    const excelRow = worksheet.addRow(row);
    excelRow.eachCell(cell => {
      cell.border = {
        top: { style: 'thin', color: { argb: 'FF000000' } },
        left: { style: 'thin', color: { argb: 'FF000000' } },
        bottom: { style: 'thin', color: { argb: 'FF000000' } },
        right: { style: 'thin', color: { argb: 'FF000000' } }
      };
    });
  });

  await workbook.xlsx.writeBuffer().then(buffer => {
    const blob = new Blob([buffer], { type: "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet" });
    const url = window.URL.createObjectURL(blob);
    const a = document.createElement("a");
    a.href = url;
    a.download = "detailed_daily.xlsx";
    document.body.appendChild(a);
    a.click();
    document.body.removeChild(a);
  });
  console.log('Excel file exported with borders.');
}

// Example usage (replace tableData with your actual data array):
const tableData = [
  ['Header', 'Col1', 'Col2'],
  ['Row1', 10, 20],
  ['Row2', 30, 40]
];

document.addEventListener("DOMContentLoaded", () => {
  // Call exportDetailedDaily when needed (for testing, uncomment below):
  // exportDetailedDaily(tableData);
});
