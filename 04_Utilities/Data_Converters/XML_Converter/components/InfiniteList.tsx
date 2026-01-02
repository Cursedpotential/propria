import React from 'react'
import { cn } from '../lib/utils'
import { useInfiniteQuery } from '../hooks/use-infinite-query'
import { SupabaseClient } from '@supabase/supabase-js'

interface InfiniteListProps {
  client: SupabaseClient | null
  tableName: string
  columns?: string
  pageSize?: number
  trailingQuery?: (query: any) => any
  renderItem: (item: any, index: number) => React.ReactNode
  className?: string
  renderNoResults?: () => React.ReactNode
  renderEndMessage?: () => React.ReactNode
  renderSkeleton?: (count: number) => React.ReactNode
}

const DefaultNoResults = () => (
  <div className="flex flex-col items-center justify-center py-10 text-center text-muted-foreground">
      <div className="bg-zinc-800/50 p-4 rounded-full mb-3">
         <svg xmlns="http://www.w3.org/2000/svg" className="h-6 w-6 text-zinc-500" fill="none" viewBox="0 0 24 24" stroke="currentColor">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M4 6h16M4 12h16M4 18h7" />
        </svg>
      </div>
      <p>No results found</p>
  </div>
)

const DefaultEndMessage = () => (
  <div className="text-center text-muted-foreground py-4 text-xs font-mono opacity-50">End of records</div>
)

const defaultSkeleton = (count: number) => (
  <div className="flex flex-col gap-2 px-4 py-2">
    {Array.from({ length: count }).map((_, index) => (
      <div key={index} className="h-10 w-full bg-white/5 animate-pulse rounded" />
    ))}
  </div>
)

export const InfiniteList: React.FC<InfiniteListProps> = ({
  client,
  tableName,
  columns = '*',
  pageSize = 20,
  trailingQuery,
  renderItem,
  className,
  renderNoResults = DefaultNoResults,
  renderEndMessage = DefaultEndMessage,
  renderSkeleton = defaultSkeleton,
}) => {
  const { data, isFetching, hasMore, fetchNextPage, isSuccess } = useInfiniteQuery({
    client,
    tableName,
    columns,
    pageSize,
    trailingQuery,
  })

  // Ref for the scrolling container
  const scrollContainerRef = React.useRef<HTMLDivElement>(null)

  // Intersection observer logic - target the last rendered *item* or a dedicated sentinel
  const loadMoreSentinelRef = React.useRef<HTMLDivElement>(null)
  const observer = React.useRef<IntersectionObserver | null>(null)

  React.useEffect(() => {
    if (observer.current) observer.current.disconnect()

    observer.current = new IntersectionObserver(
      (entries) => {
        if (entries[0].isIntersecting && hasMore && !isFetching) {
          fetchNextPage()
        }
      },
      {
        root: scrollContainerRef.current, // Use the scroll container for scroll detection
        threshold: 0.1, // Trigger when 10% of the target is visible
        rootMargin: '0px 0px 100px 0px', // Trigger loading a bit before reaching the end
      }
    )

    if (loadMoreSentinelRef.current) {
      observer.current.observe(loadMoreSentinelRef.current)
    }

    return () => {
      if (observer.current) observer.current.disconnect()
    }
  }, [isFetching, hasMore, fetchNextPage])

  return (
    <div ref={scrollContainerRef} className={cn('relative h-full overflow-auto custom-scrollbar', className)}>
      <div className="min-w-full">
        {isSuccess && data.length === 0 && renderNoResults()}

        {data.map((item, index) => renderItem(item, index))}

        {isFetching && renderSkeleton && renderSkeleton(pageSize)}

        <div ref={loadMoreSentinelRef} style={{ height: '1px' }} />

        {!hasMore && data.length > 0 && renderEndMessage()}
      </div>
    </div>
  )
}