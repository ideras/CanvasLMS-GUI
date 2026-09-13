<script>
  /**
   * DataTable — shared wrapper over TanStack Table v9 (headless).
   *
   * Owns: table instance, sorting state, <table> markup, empty/loading states.
   * Consumers own: data fetching, column definitions, per-cell content
   * (rendered through snippets via renderSnippet, which preserves the
   * consumer component's scoped CSS).
   *
   * Styling contract:
   * - Base table look comes from global styles in src/style.css (bare
   *   `table/th/td` selectors).
   * - Per-column td classes are passed via columnDef.meta.cellClass and
   *   must exist in global CSS (Svelte scoping cannot reach into this
   *   component's rendered elements).
   * - tableClass adds a hook for table variants (e.g. "compact-table").
   * - rowClass receives the raw row record and returns extra <tr> classes.
   */
  import {
    createTable,
    FlexRender,
    tableFeatures,
    rowSortingFeature,
    createSortedRowModel,
    sortFns,
    renderSnippet,
    columnSizingFeature,
    columnResizingFeature
  } from '@tanstack/svelte-table'

  let {
    data = [],
    columns,
    cells = {},             // { columnId: snippet } — snippet-backed cells, passed as props
    rowKey = null,          // fn(row) -> id, or string field name; stabilizes row identity
    sortable = true,        // master switch; columns opt out via enableSorting: false
    initialSorting = [],    // e.g. [{ id: 'score', desc: true }]
    loading = false,
    emptyMessage = 'No data.',
    tableClass = '',
    rowClass = null,        // fn(row.original) -> extra <tr> class(es)
    resizable = true        // column resize handles; columns opt out via enableResizing: false
  } = $props()

  // Columns flagged `snippet: true` get their cell content from the `cells`
  // prop (snippets cannot be referenced from script-built column defs —
  // template-scoped snippets are not visible to script closures). The
  // snippet receives the raw row record and keeps the consumer component's
  // scoped CSS.

  const features = tableFeatures({
    rowSortingFeature,
    sortedRowModel: createSortedRowModel(),
    sortFns,
    columnSizingFeature,
    columnResizingFeature
  })

  function resolveGetRowId(rk) {
    if (!rk) return undefined
    if (typeof rk === 'function') return (row, i) => String(rk(row) ?? i)
    return (row) => String(row[rk])
  }

  const resolvedColumns = columns.map((col) => {
    const colId = col.id ?? col.accessorKey
    return col.snippet && cells[colId]
      ? { ...col, id: colId, cell: ({ row }) => renderSnippet(cells[colId], row.original) }
      : col
  })

  // Sort toggles must never throw: an exception inside this delegated
  // handler aborts Svelte 5's event walk, so ancestor on:click handlers
  // (e.g. a modal overlay's close-on-click) would fire unexpectedly.
  function toggleSort(event, header) {
    try {
      header.column.getToggleSortingHandler()(event)
    } catch (err) {
      console.error('[DataTable] sort toggle failed for column', header.column.id, err)
    }
  }

  const table = createTable({
    features,
    columns: resolvedColumns,
    get data() {
      return data
    },
    initialState: { sorting: initialSorting },
    getRowId: resolveGetRowId(rowKey),
    columnResizeMode: 'onChange',
    defaultColumn: { minSize: 60 }
  })

  let theadEl
  let userResized = $state(false)
  let seeded = false
  let seededSizes = {}

  // TanStack's resize math is relative to its LOGICAL column sizes (default
  // 150px), not the browser's auto-layout widths. Seed the logical sizes
  // from the rendered layout once, right after mount, so any later drag
  // delta is relative to what the user actually sees. Widths stay fluid
  // (no explicit px) until the user actually drags a handle.
  $effect(() => {
    if (seeded || userResized || !theadEl?.rows?.[0]) return
    seeded = true
    const headers = table.getHeaderGroups()[0]?.headers ?? []
    const sizes = {}
    headers.forEach((h, i) => {
      const w = theadEl.rows[0].cells[i]?.getBoundingClientRect?.().width
      if (w) sizes[h.column.id] = Math.round(w * 100) / 100
    })
    seededSizes = sizes
    if (Object.keys(sizes).length) table.setColumnSizing((old) => ({ ...old, ...sizes }))
  })

  // dblclick reset returns the column to its seeded (as-rendered) width,
  // not TanStack's 150px default.
  function resetColumn(column) {
    const seeded = seededSizes[column.id]
    if (seeded != null) table.setColumnSizing((old) => ({ ...old, [column.id]: seeded }))
    else column.resetSize()
  }

  // Explicit widths kick in only once the user has dragged a handle; until
  // then the table keeps its browser auto-layout.
  function userWidth(column) {
    if (!userResized) return ''
    return `width: ${column.getSize()}px`
  }

  function beginResize(event, header) {
    userResized = true
    header.getResizeHandler()(event)
  }

  let rows = $derived(table.getRowModel().rows)
  let columnCount = $derived(table.getAllLeafColumns().length)
</script>

<table class={tableClass} style={userResized ? 'table-layout: fixed; width: 100%' : ''}>
  <thead bind:this={theadEl}>
    {#each table.getHeaderGroups() as headerGroup (headerGroup.id)}
      <tr>
        {#each headerGroup.headers as header (header.id)}
          {@const canSort = sortable && header.column.getCanSort()}
          <!-- svelte-ignore a11y_click_events_have_key_events -->
          <!-- svelte-ignore a11y_no_static_element_interactions -->
          <th
            class={[canSort ? 'sortable-th' : '', header.column.columnDef.meta?.headerClass ?? '']}
            style={userWidth(header.column)}
            onclick={canSort ? (event) => toggleSort(event, header) : undefined}
          >
            {#if !header.isPlaceholder}
              <FlexRender {header} />
              {#if canSort}
                {#if header.column.getIsSorted() === 'asc'}
                  <span class="sort-indicator">▲</span>
                {:else if header.column.getIsSorted() === 'desc'}
                  <span class="sort-indicator">▼</span>
                {/if}
              {/if}
            {/if}
            {#if resizable && header.column.getCanResize()}
              <!-- stopPropagation on click: the resizer must not trigger the
                   th's sort toggle; dblclick resets the column size -->
              <div
                class="col-resizer"
                class:resizing={header.column.getIsResizing()}
                title="Drag to resize · double-click to reset"
                onmousedown={(e) => beginResize(e, header)}
                ontouchstart={(e) => beginResize(e, header)}
                ondblclick={() => resetColumn(header.column)}
                onclick={(e) => e.stopPropagation()}
              ></div>
            {/if}
          </th>
        {/each}
      </tr>
    {/each}
  </thead>
  <tbody>
    {#if loading}
      <tr>
        <td class="data-table-status" colspan={columnCount}>Loading…</td>
      </tr>
    {:else if rows.length === 0}
      <tr>
        <td class="data-table-status" colspan={columnCount}>{emptyMessage}</td>
      </tr>
    {:else}
      {#each rows as row (row.id)}
        <tr class={rowClass ? rowClass(row.original) : ''}>
          {#each row.getAllCells() as cell (cell.id)}
            <td class={cell.column.columnDef.meta?.cellClass ?? ''} style={userWidth(cell.column)}>
              <FlexRender cell={cell} />
            </td>
          {/each}
        </tr>
      {/each}
    {/if}
  </tbody>
</table>

<style>
  th {
    position: relative;
  }

  .col-resizer {
    position: absolute;
    top: 0;
    right: 0;
    height: 100%;
    width: 9px;
    cursor: col-resize;
    touch-action: none;
    user-select: none;
    z-index: 2;
  }

  .col-resizer.resizing {
    background: var(--frost-blue);
    opacity: 0.35;
  }

  .sortable-th {
    cursor: pointer;
    user-select: none;
  }

  .sortable-th:hover {
    color: var(--frost-blue);
  }

  .sort-indicator {
    font-size: 9px;
    margin-left: 4px;
  }

  .data-table-status {
    text-align: center;
    padding: 32px;
    color: var(--text-secondary);
  }
</style>