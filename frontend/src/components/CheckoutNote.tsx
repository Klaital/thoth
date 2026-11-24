import { Link } from "react-router-dom";
import { CardCheckout } from "../api/types";

// "2 in Atraxa": copies of a collection card that are checked out to decks.
export default function CheckoutNote({ checkouts }: { checkouts?: CardCheckout[] }) {
  if (!checkouts || checkouts.length === 0) return null;
  return (
    <span className="item-meta card-checkouts">
      {checkouts.map((c, i) => (
        <span key={c.deckId}>
          {i > 0 && " · "}
          {c.quantity} in <Link to={`/decks/${c.deckId}`}>{c.deckName}</Link>
        </span>
      ))}
    </span>
  );
}
