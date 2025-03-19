document.getElementById("tableFilter").addEventListener("input", function () {
    const filter = this.value.toLowerCase();
    const rows = document.querySelectorAll("#dataTable tbody tr");

    for (let i = 0; i < rows.length; ++i) {
        let text = rows[i].textContent.toLowerCase();
        rows[i].style.display = text.indexOf(filter) > -1 ? "" : "none";
    }
});

document.querySelectorAll("#dataTable th").forEach((headerCell) => {
    headerCell.addEventListener("click", () => {
        const columnIndex = Array.from(headerCell.parentNode.children).indexOf(
            headerCell
        );
        const isAscending = headerCell.classList.contains("th-sort-asc");

        sortTableByColumn(columnIndex, !isAscending);
    });
});

function sortTableByColumn(column, asc) {
    const table = document.getElementById("dataTable");
    const tbody = table.querySelector("tbody");
    const rows = Array.from(tbody.querySelectorAll("tr"));

    table.querySelectorAll("th").forEach((th) => {
        th.classList.remove("th-sort-asc", "th-sort-desc");
    });

    table
        .querySelectorAll("th")
        [column].classList.add(asc ? "th-sort-asc" : "th-sort-desc");

    const sortedRows = rows.sort((a, b) => {
        const aValue = a
            .querySelector(`td:nth-child(${column + 1})`)
            .textContent.trim();
        const bValue = b
            .querySelector(`td:nth-child(${column + 1})`)
            .textContent.trim();

        return (
            aValue.localeCompare(bValue, undefined, { numeric: true }) *
            (asc ? 1 : -1)
        );
    });

    while (tbody.firstChild) {
        tbody.removeChild(tbody.firstChild);
    }

    tbody.append(...sortedRows);
}
