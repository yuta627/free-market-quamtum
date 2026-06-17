import { Link } from "react-router-dom";
import { Heart } from "lucide-react";
import { type Product } from "../api/products";
import styles from "./ProductCard.module.css";

function parseFirstImage(raw: string): string | null {
  try {
    const arr = JSON.parse(raw) as string[];
    return arr[0] ?? null;
  } catch {
    return null;
  }
}

interface ProductCardProps {
  product: Product;
  liked?: boolean;
  onToggleLike?: (productId: number) => void;
  statusBadge?: string;
}

export default function ProductCard({ product, liked, onToggleLike, statusBadge }: ProductCardProps) {
  const image = parseFirstImage(product.image_urls);

  return (
    <Link to={`/products/${product.id}`} className={styles.card}>
      <div className={styles.imageWrap}>
        {image ? (
          <img src={image} alt={product.title} className={styles.image} />
        ) : (
          <div className={styles.noImagePlaceholder}>No Image</div>
        )}
        {onToggleLike && (
          <button
            className={styles.likeBtn}
            aria-label={liked ? "いいねを取り消す" : "いいねする"}
            onClick={(e) => {
              e.preventDefault();
              e.stopPropagation();
              onToggleLike(product.id);
            }}
          >
            <Heart size={18} fill={liked ? "#f60" : "none"} color={liked ? "#f60" : "#888"} />
          </button>
        )}
        {statusBadge && <span className={styles.statusBadge}>{statusBadge}</span>}
      </div>
      <div className={styles.cardBody}>
        <p className={styles.cardTitle}>{product.title}</p>
        <p className={styles.cardPrice}>¥{product.price.toLocaleString()}</p>
      </div>
    </Link>
  );
}
