interface Props {
  id: string;
  quantity: number;
  // Non-owners see the quantity without controls.
  editable: boolean;
  onQuantityChange: (id: string, delta: 1 | -1) => void;
}

export default function QuantityStepper({ id, quantity, editable, onQuantityChange }: Props) {
  if (!editable) {
    return <span className="item-qty">×{quantity}</span>;
  }
  return (
    <div className="qty-stepper">
      <button
        className="btn-ghost"
        onClick={() => onQuantityChange(id, -1)}
        disabled={quantity <= 1}
        aria-label="Decrease quantity"
      >−</button>
      <span className="item-qty">{quantity}</span>
      <button
        className="btn-ghost"
        onClick={() => onQuantityChange(id, 1)}
        aria-label="Increase quantity"
      >+</button>
    </div>
  );
}
