// Byline: Claude Code · Sonnet · 2026-10-02
import { useParams } from "@tanstack/react-router";

import { ReviewDetailView } from "@/components/mobile/review-views";

export default function MobileReviewDetailPage() {
  const { handle } = useParams({ strict: false }) as { handle?: string };
  return handle ? <ReviewDetailView handle={handle} /> : null;
}
