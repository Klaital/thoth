import { CollectionCommander } from "../api/types";

interface Props {
  commanders: CollectionCommander[];
  size?: "small" | "large";
}

// Shows a deck's commander card(s) with their names, e.g. under the collection name.
export default function CommanderSummary({ commanders, size = "small" }: Props) {
  return (
    <div className={`commander-summary commander-summary-${size}`}>
      <div className="commander-images">
        {commanders.map((c) =>
          c.imageUri ? <img key={c.cardName} src={c.imageUri} alt={c.cardName} /> : null,
        )}
      </div>
      <span className="commander-names">{commanders.map((c) => c.cardName).join(" & ")}</span>
    </div>
  );
}
