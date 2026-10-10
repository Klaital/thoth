import { useParams, Link } from "react-router-dom";

// This page is a placeholder; the actual form rendered depends on collection type.
// Once the collection detail is fetched, it will conditionally render
// AddCardItemForm, AddSealedItemForm, or AddBookDvdItemForm.

export default function AddItemPage() {
  const { id } = useParams<{ id: string }>();

  return (
    <main>
      <Link to={`/collections/${id}`}>← Back</Link>
      <h1>Add Item</h1>
      <p>TODO: render item form based on collection type.</p>
    </main>
  );
}
