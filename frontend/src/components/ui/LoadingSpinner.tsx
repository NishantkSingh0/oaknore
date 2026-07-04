import { clsx } from 'clsx'

interface Props { fullScreen?: boolean; size?: 'sm' | 'md' | 'lg' }

export default function LoadingSpinner({ fullScreen, size = 'md' }: Props) {
  const s = { sm: 'h-4 w-4 border-2', md: 'h-8 w-8 border-2', lg: 'h-12 w-12 border-[3px]' }[size]
  const spinner = (
    <span className={clsx('animate-spin rounded-full border-brand-600 border-t-transparent', s)} />
  )
  if (fullScreen)
    return <div className="flex h-screen items-center justify-center">{spinner}</div>
  return spinner
}
