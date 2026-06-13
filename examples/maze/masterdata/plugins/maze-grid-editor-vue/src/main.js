import { createApp, computed, h, reactive, ref } from "vue";
import "./styles.css";

const TERRAIN = ["wall", "path", "start", "goal"];

function keyString(key) {
  return typeof key === "object" && key ? `${key.map_id}:${key.x}:${key.y}` : String(key);
}

function cellRecord(mapId, cell) {
  const data = { terrain: cell.terrain };
  if (cell.enemy_id) data.enemy_id = cell.enemy_id;
  return {
    key: { map_id: mapId, x: cell.x, y: cell.y },
    data
  };
}

function mazeChangeSet(mapId, cells) {
  return {
    tables: {
      maze_cell: {
        updates: cells.map((cell) => ({
          previousKey: { map_id: mapId, x: cell.x, y: cell.y },
          record: cellRecord(mapId, cell)
        }))
      }
    }
  };
}

function createMazeEditor(context) {
  const mapRecord = context.tables?.maze_map?.records?.[0];
  const mapId = mapRecord?.key ?? mapRecord?.data?.map_id ?? "";
  const width = Number(mapRecord?.data?.width ?? 10);
  const height = Number(mapRecord?.data?.height ?? 10);
  const sourceCells = context.tables?.maze_cell?.records ?? [];
  const cellByKey = new Map(sourceCells.map((record) => [keyString(record.key), record]));
  const cells = reactive([]);
  for (let y = 0; y < height; y += 1) {
    for (let x = 0; x < width; x += 1) {
      const record = cellByKey.get(`${mapId}:${x}:${y}`);
      cells.push({
        x,
        y,
        terrain: record?.data?.terrain ?? "wall",
        enemy_id: record?.data?.enemy_id ?? ""
      });
    }
  }
  const selectedTerrain = ref("wall");
  const selectedEnemy = ref("");
  const message = ref("");
  const enemies = computed(() => context.tables?.enemy?.records ?? []);
  const selectedCell = ref(null);
  const dragging = ref(false);
  const dragChanged = ref(false);

  function propose() {
    context.host?.setDirty?.(true);
    window.__mazeGridEditorLastChangeSet = mazeChangeSet(mapId, cells);
    if (context.host?.proposeChanges) {
      context.host.proposeChanges(window.__mazeGridEditorLastChangeSet)
        .then(() => {
          message.value = "Change proposed.";
        })
        .catch((error) => {
          message.value = error.message ?? "Failed to propose change.";
        });
    }
  }

  function applyTerrain(cell) {
    if (!cell) return false;
    const nextTerrain = selectedTerrain.value;
    const nextEnemyId = nextTerrain === "wall" ? "" : cell.enemy_id;
    if (cell.terrain === nextTerrain && cell.enemy_id === nextEnemyId) return false;
    cell.terrain = nextTerrain;
    cell.enemy_id = nextEnemyId;
    selectedCell.value = cell;
    return true;
  }

  function paint(cell) {
    if (!applyTerrain(cell)) return;
    propose();
  }

  function beginPaint(cell, event) {
    if (event.button !== 0) return;
    event.preventDefault();
    dragging.value = true;
    dragChanged.value = applyTerrain(cell);
    selectedCell.value = cell;
  }

  function dragPaint(cell, event) {
    if (!dragging.value) return;
    event.preventDefault();
    if (applyTerrain(cell)) dragChanged.value = true;
  }

  function finishPaint() {
    if (!dragging.value) return;
    dragging.value = false;
    if (dragChanged.value) propose();
    dragChanged.value = false;
  }

  function keyboardPaint(cell, event) {
    if (event.detail !== 0) return;
    paint(cell);
  }

  function placeEnemy(cell) {
    if (!cell) return;
    if (cell.terrain === "wall") cell.terrain = "path";
    cell.enemy_id = selectedEnemy.value;
    propose();
  }

  const app = createApp({
    setup() {
      return {
        mapRecord,
        mapId,
        width,
        height,
        cells,
        TERRAIN,
        selectedTerrain,
        selectedEnemy,
        selectedCell,
        enemies,
        message,
        dragging,
        paint,
        beginPaint,
        dragPaint,
        finishPaint,
        keyboardPaint,
        placeEnemy
      };
    },
    render() {
      return h("main", { class: "shell" }, [
        h("aside", { class: "palette" }, [
          h("span", { class: "eyebrow" }, "Maze"),
          h("h1", {}, this.mapRecord?.data?.name || this.mapId),
          h("div", { class: "terrainButtons" }, this.TERRAIN.map((terrain) => h("button", {
            key: terrain,
            type: "button",
            class: { selected: this.selectedTerrain === terrain },
            onClick: () => {
              this.selectedTerrain = terrain;
            }
          }, terrain))),
          h("label", {}, [
            "Enemy placement",
            h("select", {
              value: this.selectedEnemy,
              onChange: (event) => {
                this.selectedEnemy = event.target.value;
                this.placeEnemy(this.selectedCell);
              }
            }, [
              h("option", { value: "" }, "None"),
              ...this.enemies.map((enemy) => h("option", { key: enemy.key, value: enemy.key }, enemy.data?.name || enemy.name || enemy.key))
            ])
          ]),
          h("p", {}, this.message)
        ]),
        h("section", {
          class: "grid",
          style: { gridTemplateColumns: `repeat(${this.width}, minmax(0, 1fr))` },
          onPointerup: this.finishPaint,
          onPointercancel: this.finishPaint,
          onPointerleave: this.finishPaint
        }, this.cells.map((cell) => h("button", {
          key: `${cell.x}-${cell.y}`,
          type: "button",
          class: ["cell", cell.terrain, { active: this.selectedCell === cell, enemy: cell.enemy_id }],
          onPointerdown: (event) => this.beginPaint(cell, event),
          onPointerenter: (event) => this.dragPaint(cell, event),
          onClick: (event) => this.keyboardPaint(cell, event),
          onContextmenu: (event) => {
            event.preventDefault();
            this.selectedCell = cell;
            this.placeEnemy(cell);
          }
        }, cell.terrain === "start" ? "S" : (cell.terrain === "goal" ? "G" : (cell.enemy_id ? "E" : "")))))
      ]);
    }
  });

  return {
    mount() {
      app.mount("#plugin-root");
    },
    unmount() {
      app.unmount();
    },
    beforeSave() {
      return window.__mazeGridEditorLastChangeSet ?? mazeChangeSet(mapId, cells);
    }
  };
}

window.MasterDataMatePlugin = async (context) => {
  const editor = createMazeEditor(context);
  editor.mount();
  return {
    beforeSave: editor.beforeSave,
    dispose: editor.unmount,
    onHostDataReloaded: () => {}
  };
};
