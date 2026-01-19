import type { ReactNode } from 'react'
import { cn } from '@/gen/shadcn/lib/util'

interface ContentPanelProps {
    children: ReactNode
    className?: string
    contentClassName?: string
    variant?: 'default' | 'error'
}

export const ContentPanel = ({
    children,
    className,
    contentClassName,
    variant = 'default',
}: ContentPanelProps) => {
    const variantStyles = {
        default: {
            border: 'border-zinc-800/50',
            bg: 'bg-zinc-950/50',
            text: 'text-zinc-200',
        },
        error: {
            border: 'border-red-500/20',
            bg: 'bg-red-950/20',
            text: 'text-red-300',
        },
    }

    const styles = variantStyles[variant]

    return (
        <div
            className={cn(
                'rounded-lg border p-4',
                styles.border,
                styles.bg,
                className
            )}
        >
            <pre
                className={cn(
                    'whitespace-pre-wrap break-all font-mono text-sm leading-relaxed',
                    styles.text,
                    contentClassName
                )}
            >
                {children}
            </pre>
        </div>
    )
}
