import { SupabaseClient } from '@supabase/supabase-js'
import { useEffect, useMemo, useSyncExternalStore } from 'react'

// Types adjusted to be generic since we don't have the generated types file locally available in this context
// In a real setup, these would be imported from database.types.ts
type SupabaseTableName = string
type SupabaseTableData = any

interface UseInfiniteQueryProps {
  client: SupabaseClient | null
  tableName: SupabaseTableName
  columns?: string
  pageSize?: number
  trailingQuery?: (query: any) => any
}

interface StoreState<TData> {
  data: TData[]
  count: number
  isSuccess: boolean
  isLoading: boolean
  isFetching: boolean
  error: Error | null
  hasInitialFetch: boolean
}

type Listener = () => void

function createStore<TData>(props: UseInfiniteQueryProps) {
  const { client, tableName, columns = '*', pageSize = 20, trailingQuery } = props

  let state: StoreState<TData> = {
    data: [],
    count: 0,
    isSuccess: false,
    isLoading: false,
    isFetching: false,
    error: null,
    hasInitialFetch: false,
  }

  const listeners = new Set<Listener>()

  const notify = () => {
    listeners.forEach((listener) => listener())
  }

  const setState = (newState: Partial<StoreState<TData>>) => {
    state = { ...state, ...newState }
    notify()
  }

  const fetchPage = async (skip: number) => {
    if (!client) return
    if (state.hasInitialFetch && (state.isFetching || state.count <= state.data.length)) return

    setState({ isFetching: true })

    let query = client
      .from(tableName)
      .select(columns, { count: 'exact' })

    if (trailingQuery) {
      query = trailingQuery(query)
    }
    
    // Sort logic depends on table typically, but let's assume standard ISO date sort if not provided
    // or let trailingQuery handle it. For now, we rely on trailingQuery or default order.
    
    const { data: newData, count, error } = await query.range(skip, skip + pageSize - 1)

    if (error) {
      console.error('An unexpected error occurred:', error)
      setState({ error: error as any })
    } else {
      setState({
        data: [...state.data, ...(newData as TData[])],
        count: count || 0,
        isSuccess: true,
        error: null,
      })
    }
    setState({ isFetching: false })
  }

  const fetchNextPage = async () => {
    if (state.isFetching) return
    await fetchPage(state.data.length)
  }

  const initialize = async () => {
    setState({ isLoading: true, isSuccess: false, data: [] })
    await fetchNextPage()
    setState({ isLoading: false, hasInitialFetch: true })
  }

  return {
    getState: () => state,
    subscribe: (listener: Listener) => {
      listeners.add(listener)
      return () => { listeners.delete(listener) }
    },
    fetchNextPage,
    initialize,
  }
}

const initialState: any = {
  data: [],
  count: 0,
  isSuccess: false,
  isLoading: false,
  isFetching: false,
  error: null,
  hasInitialFetch: false,
}

export function useInfiniteQuery<TData = any>(props: UseInfiniteQueryProps) {
  // Use useMemo to ensure the store is created synchronously when dependencies change.
  // This avoids the "render with old store -> useEffect -> update ref" lag and ensures
  // useSyncExternalStore subscribes to the correct store instance.
  // CRITICAL: props.trailingQuery MUST be memoized by the caller.
  const store = useMemo(() => {
    return createStore<TData>(props)
  }, [
    props.client,
    props.tableName,
    props.columns,
    props.pageSize,
    props.trailingQuery
  ])

  // Initial fetch
  useEffect(() => {
    if (!store.getState().hasInitialFetch && props.client) {
      store.initialize()
    }
  }, [store, props.client])

  const state = useSyncExternalStore(
    store.subscribe,
    store.getState,
    () => initialState as StoreState<TData>
  )

  return {
    data: state.data,
    count: state.count,
    isSuccess: state.isSuccess,
    isLoading: state.isLoading,
    isFetching: state.isFetching,
    error: state.error,
    hasMore: state.count > state.data.length,
    fetchNextPage: store.fetchNextPage,
  }
}