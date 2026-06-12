import React, { useEffect, useMemo, useState } from "react";
import { createRoot } from "react-dom/client";
import "./styles.css";

const STAT_FIELDS = ["hp", "attack", "defense", "speed", "magic", "luck"];

function recordKey(record) {
  return record?.key ?? record?.data?.enemy_id ?? "";
}

function canonicalRecord(key, draft) {
  return {
    key,
    name: draft.name,
    data: {
      name: draft.name,
      image: draft.image || undefined,
      hp: Number(draft.hp) || 0,
      attack: Number(draft.attack) || 0,
      defense: Number(draft.defense) || 0,
      speed: Number(draft.speed) || 0,
      magic: Number(draft.magic) || 0,
      luck: Number(draft.luck) || 0
    }
  };
}

function enemyChangeSet(key, draft) {
  return {
    tables: {
      enemy: {
        updates: [
          {
            previousKey: key,
            record: canonicalRecord(key, draft)
          }
        ]
      }
    }
  };
}

function radarPoints(draft) {
  const center = 92;
  const maxRadius = 72;
  return STAT_FIELDS.map((field, index) => {
    const angle = (Math.PI * 2 * index) / STAT_FIELDS.length - Math.PI / 2;
    const raw = Math.max(0, Math.min(100, Number(draft[field]) || 0));
    const radius = (raw / 100) * maxRadius;
    return `${center + Math.cos(angle) * radius},${center + Math.sin(angle) * radius}`;
  }).join(" ");
}

function axisLine(index) {
  const center = 92;
  const radius = 72;
  const angle = (Math.PI * 2 * index) / STAT_FIELDS.length - Math.PI / 2;
  return {
    x1: center,
    y1: center,
    x2: center + Math.cos(angle) * radius,
    y2: center + Math.sin(angle) * radius
  };
}

function EnemyEditor({ context }) {
  const host = context.host ?? {};
  const record = context.tables?.enemy?.records?.[0];
  const key = recordKey(record);
  const initial = useMemo(() => ({
    name: record?.data?.name ?? record?.name ?? String(key),
    image: record?.data?.image ?? null,
    hp: record?.data?.hp ?? 0,
    attack: record?.data?.attack ?? 0,
    defense: record?.data?.defense ?? 0,
    speed: record?.data?.speed ?? 0,
    magic: record?.data?.magic ?? 0,
    luck: record?.data?.luck ?? 0
  }), [record, key]);
  const [draft, setDraft] = useState(initial);
  const [imageUrl, setImageUrl] = useState("");
  const [message, setMessage] = useState("");

  useEffect(() => {
    let alive = true;
    if (!draft.image || !host.getBinaryAssetUrl) {
      setImageUrl("");
      return;
    }
    Promise.resolve(host.getBinaryAssetUrl({ table: "enemy", key, field: "image" }))
      .then((url) => {
        if (alive) setImageUrl(url);
      })
      .catch(() => {
        if (alive) setImageUrl("");
      });
    return () => {
      alive = false;
    };
  }, [draft.image, host, key]);

  async function propose(next) {
    setDraft(next);
    host.setDirty?.(true);
    if (host.proposeChanges) {
      try {
        await host.proposeChanges(enemyChangeSet(key, next));
        setMessage("Change proposed.");
      } catch (error) {
        setMessage(error.message ?? "Failed to propose change.");
      }
    }
  }

  async function uploadImage(file) {
    if (!file || !host.uploadBinaryAsset) return;
    try {
      const result = await host.uploadBinaryAsset({ table: "enemy", key, field: "image", file });
      const metadata = result?.metadata ?? result;
      await propose({ ...draft, image: metadata });
      setMessage(`Uploaded ${metadata?.original_name ?? file.name}.`);
    } catch (error) {
      setMessage(error.message ?? "Image upload failed.");
    }
  }

  window.__enemyStatusEditorLastChangeSet = enemyChangeSet(key, draft);

  if (!record || context.entry?.kind !== "record") {
    return <main className="shell"><p>Open this plugin from one enemy record.</p></main>;
  }

  return (
    <main className="shell">
      <section className="preview">
        <div className="imageDrop" onDragOver={(event) => event.preventDefault()} onDrop={(event) => {
          event.preventDefault();
          uploadImage(event.dataTransfer.files?.[0]);
        }}>
          {imageUrl ? <img src={imageUrl} alt={draft.name} /> : <span>No image</span>}
        </div>
        <label className="uploadButton">
          Upload image
          <input type="file" accept="image/png,image/jpeg,image/gif" onChange={(event) => uploadImage(event.target.files?.[0])} />
        </label>
      </section>

      <section className="editor">
        <header>
          <span className="eyebrow">Enemy</span>
          <input value={draft.name} onChange={(event) => propose({ ...draft, name: event.target.value })} />
        </header>
        <div className="statGrid">
          {STAT_FIELDS.map((field) => (
            <label key={field}>
              <span>{field}</span>
              <input type="number" min="0" max="100" value={draft[field]} onChange={(event) => propose({ ...draft, [field]: Number(event.target.value) })} />
            </label>
          ))}
        </div>
        <p className="message">{message}</p>
      </section>

      <section className="radar">
        <svg viewBox="0 0 184 184" role="img" aria-label="Enemy status radar chart">
          {[24, 48, 72].map((radius) => <circle key={radius} cx="92" cy="92" r={radius} />)}
          {STAT_FIELDS.map((field, index) => {
            const line = axisLine(index);
            return <line key={field} {...line} />;
          })}
          <polygon points={radarPoints(draft)} />
        </svg>
      </section>
    </main>
  );
}

window.MasterDataMatePlugin = async (context) => {
  const rootElement = document.getElementById("plugin-root");
  const root = createRoot(rootElement);
  root.render(<EnemyEditor context={context} />);
  return {
    beforeSave: () => {
      const record = context.tables?.enemy?.records?.[0];
      return window.__enemyStatusEditorLastChangeSet ?? enemyChangeSet(recordKey(record), record?.data ?? {});
    },
    dispose: () => root.unmount(),
    onHostDataReloaded: (nextContext) => root.render(<EnemyEditor context={nextContext} />)
  };
};

