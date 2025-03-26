const ExcelJS = require('exceljs');

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

  await workbook.xlsx.writeFile('detailed_daily.xlsx');
  console.log('Excel file exported with borders.');
}

// Example usage (replace tableData with your actual data array):
const tableData = [
  ['Header', 'Col1', 'Col2'],
  ['Row1', 10, 20],
  ['Row2', 30, 40]
];

exportDetailedDaily(tableData);
