/**
 * Простой скрипт для выделения столбцов таблицы
 */
document.addEventListener('DOMContentLoaded', function() {
    // Функция для обработки клика по ячейке
    function handleTableClick(e) {
        // Находим ячейку, по которой кликнули
        const cell = e.target.closest('td, th');
        if (!cell) return;
        
        // Находим таблицу
        const table = cell.closest('table');
        if (!table) return;
        
        // Определяем индекс столбца
        const columnIndex = Array.from(cell.parentNode.children).indexOf(cell);
        
        // Создаем диапазон для выделения
        const range = document.createRange();
        
        // Находим все ячейки в этом столбце
        const cells = [];
        for (let i = 0; i < table.rows.length; i++) {
            if (table.rows[i].cells[columnIndex]) {
                cells.push(table.rows[i].cells[columnIndex]);
            }
        }
        
        if (cells.length === 0) return;
        
        // Устанавливаем начало и конец диапазона
        range.setStartBefore(cells[0]);
        range.setEndAfter(cells[cells.length - 1]);
        
        // Очищаем текущее выделение и добавляем новое
        const selection = window.getSelection();
        selection.removeAllRanges();
        selection.addRange(range);
    }
    
    // Функция для добавления обработчиков к таблицам
    function setupTables() {
        const tables = document.querySelectorAll('table');
        tables.forEach(table => {
            // Удаляем старый обработчик, если он есть
            table.removeEventListener('click', handleTableClick);
            // Добавляем новый обработчик
            table.addEventListener('click', handleTableClick);
        });
    }
    
    // Инициализация при загрузке страницы
    setupTables();
    
    // Наблюдатель для обработки динамически добавленных таблиц
    const observer = new MutationObserver(mutations => {
        let shouldSetup = false;
        
        mutations.forEach(mutation => {
            if (mutation.addedNodes && mutation.addedNodes.length) {
                for (let i = 0; i < mutation.addedNodes.length; i++) {
                    const node = mutation.addedNodes[i];
                    if (node.nodeName === 'TABLE' || 
                        (node.nodeType === 1 && node.querySelector && node.querySelector('table'))) {
                        shouldSetup = true;
                        break;
                    }
                }
            }
        });
        
        if (shouldSetup) {
            setupTables();
        }
    });
    
    observer.observe(document.body, { childList: true, subtree: true });
}); 