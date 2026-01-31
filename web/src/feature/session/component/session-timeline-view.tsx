import { useMemo, useRef, useState, useCallback, useEffect } from 'react'
import { Card, CardContent, CardHeader, CardTitle } from '@/gen/shadcn/ui/card'
import { Badge } from '@/gen/shadcn/ui/badge'
import { Button } from '@/gen/shadcn/ui/button'
import { Slider } from '@/gen/shadcn/ui/slider'
import { Network, ZoomIn, ZoomOut } from 'lucide-react'
import type { TimelineSegment } from '../type/graph'
import type { TimelineData } from '../util/graph-data'
import { timestampToX } from '../util/graph-data'

// Constants for layout
const LANE_HEIGHT = 60
const PADDING_Y = 80
const SEGMENT_LINE_WIDTH = 4
const ENDPOINT_RADIUS = 6
const MIN_SVG_WIDTH = 400

// Zoom scale range (duration that fits in viewport)
const MIN_DURATION_FIT = 30 // Zoom in max: 30 seconds fits in viewport
const MAX_DURATION_FIT = 3 * 3600 // Zoom out max: 3 hours fits in viewport

interface SessionTimelineViewProps {
  timelineData: TimelineData
  onSegmentClick: (segment: TimelineSegment) => void
  selectedSegmentId?: string
}

// Color configuration for segments
const segmentColors = {
  main: { line: '#8b5cf6', point: '#a78bfa', hover: '#c4b5fd' }, // violet
  sub: { line: '#06b6d4', point: '#22d3ee', hover: '#67e8f9' }, // cyan
}

// SessionTimelineView renders a segment-based timeline of agent sessions
export const SessionTimelineView = ({
  timelineData,
  onSegmentClick,
  selectedSegmentId,
}: SessionTimelineViewProps) => {
  const containerRef = useRef<HTMLDivElement>(null)
  const [hoveredSegment, setHoveredSegment] = useState<TimelineSegment | null>(
    null,
  )
  const [containerWidth, setContainerWidth] = useState(800)

  // Use the provided timeline data directly
  const segmentData = timelineData

  // Measure container width
  useEffect(() => {
    if (!containerRef.current) return

    const observer = new ResizeObserver((entries) => {
      setContainerWidth(entries[0].contentRect.width)
    })

    observer.observe(containerRef.current)
    return () => observer.disconnect()
  }, [])

  // zoomLevel 1.0 = current session fits in viewport
  // zoomLevel > 1.0 = zoom in (shorter duration fits in viewport)
  // zoomLevel < 1.0 = zoom out (longer duration fits in viewport)
  const [zoomLevel, setZoomLevel] = useState(1.0)

  // Zoom range calculation: determine min/max based on session duration
  const { minZoom, maxZoom } = useMemo(() => {
    const duration = segmentData.totalDuration || 1
    // Zoom in max: zoom level when 30 seconds fits in viewport
    const maxZ = duration / MIN_DURATION_FIT
    // Zoom out max: zoom level when 3 hours fits in viewport
    const minZ = duration / MAX_DURATION_FIT
    return { minZoom: Math.max(0.01, minZ), maxZoom: Math.max(1, maxZ) }
  }, [segmentData.totalDuration])

  // Calculate SVG dimensions
  const { svgWidth, svgHeight, centerY } = useMemo(() => {
    // Width based on zoom level
    // zoomLevel 1.0 → containerWidth (session fits in viewport)
    // zoomLevel 2.0 → containerWidth * 2 (scroll needed)
    const contentWidth = (containerWidth - 80) * zoomLevel
    const width = Math.max(MIN_SVG_WIDTH, contentWidth + 80)

    // Height based on number of segments
    const maxY = Math.max(
      ...segmentData.segments.map((s) => Math.abs(s.yPosition)),
      0,
    )
    const height = maxY * 2 + PADDING_Y * 2 + LANE_HEIGHT

    // Center Y position (where main lane sits)
    const center = height / 2

    return { svgWidth: width, svgHeight: height, centerY: center }
  }, [segmentData, containerWidth, zoomLevel])

  // Count SubAgent segments
  const subAgentCount = segmentData.segments.filter(
    (s) => s.id !== 'main',
  ).length

  // Format timestamp for tooltip
  const formatTime = useCallback(
    (seconds: number): string => {
      const startSeconds = segmentData.timeRange.start
      const elapsed = seconds - startSeconds
      const mins = Math.floor(elapsed / 60)
      const secs = Math.floor(elapsed % 60)
      return `${mins}:${secs.toString().padStart(2, '0')}`
    },
    [segmentData.timeRange.start],
  )

  // Handle segment click
  const handleSegmentClick = useCallback(
    (segment: TimelineSegment) => {
      onSegmentClick(segment)
    },
    [onSegmentClick],
  )

  // Get colors for a segment
  const getSegmentColors = (segment: TimelineSegment) => {
    return segment.id === 'main' ? segmentColors.main : segmentColors.sub
  }

  return (
    <Card className="group relative flex min-h-[400px] min-w-0 flex-col overflow-hidden border-zinc-800/50 bg-zinc-900/80 backdrop-blur-sm transition-all duration-300 hover:border-zinc-700/50">
      {/* Ambient glow */}
      <div className="pointer-events-none absolute -right-16 -top-16 h-32 w-32 rounded-full bg-violet-500/5 blur-3xl transition-opacity duration-500 group-hover:bg-violet-500/10" />

      <CardHeader className="flex-shrink-0 border-b border-zinc-800/50 pb-4">
        <div className="flex items-center justify-between">
          <div className="flex items-center gap-3">
            <div className="rounded-lg border border-violet-500/20 bg-violet-500/10 p-2">
              <Network className="h-4 w-4 text-violet-400" />
            </div>
            <div>
              <CardTitle className="text-sm font-semibold text-zinc-100">
                Session Timeline
              </CardTitle>
              <p className="font-mono text-[10px] text-zinc-600">
                Agent session segments
              </p>
            </div>
          </div>
          <div className="flex items-center gap-3">
            {/* Zoom Controls */}
            <div className="flex items-center gap-1 rounded-md border border-zinc-700/50 bg-zinc-800/50 px-1.5 py-1">
              <Button
                variant="ghost"
                size="icon"
                className="h-6 w-6"
                onClick={() => setZoomLevel(Math.max(minZoom, zoomLevel / 1.5))}
                disabled={zoomLevel <= minZoom}
              >
                <ZoomOut className="h-3.5 w-3.5" />
              </Button>

              <Slider
                value={[Math.log(zoomLevel)]}
                min={Math.log(minZoom)}
                max={Math.log(maxZoom)}
                step={0.1}
                onValueChange={([v]) => setZoomLevel(Math.exp(v))}
                className="w-20"
              />

              <Button
                variant="ghost"
                size="icon"
                className="h-6 w-6"
                onClick={() => setZoomLevel(Math.min(maxZoom, zoomLevel * 1.5))}
                disabled={zoomLevel >= maxZoom}
              >
                <ZoomIn className="h-3.5 w-3.5" />
              </Button>

              <Button
                variant="ghost"
                size="sm"
                className="h-6 px-2 text-xs"
                onClick={() => setZoomLevel(1.0)}
              >
                Fit
              </Button>

              <span className="min-w-[3.5rem] px-1 text-right font-mono text-[10px] text-zinc-500">
                {Math.round(zoomLevel * 100)}%
              </span>
            </div>

            {/* Segment Legend */}
            <div className="flex items-center gap-4">
              <div className="flex items-center gap-1.5">
                <svg width={24} height={16}>
                  <line
                    x1={4}
                    y1={8}
                    x2={20}
                    y2={8}
                    stroke={segmentColors.main.line}
                    strokeWidth={3}
                  />
                  <circle
                    cx={4}
                    cy={8}
                    r={3}
                    fill={segmentColors.main.point}
                  />
                  <circle
                    cx={20}
                    cy={8}
                    r={3}
                    fill={segmentColors.main.point}
                  />
                </svg>
                <span className="font-mono text-[10px] text-zinc-500">
                  Main
                </span>
              </div>
              <div className="flex items-center gap-1.5">
                <svg width={24} height={16}>
                  <line
                    x1={4}
                    y1={8}
                    x2={20}
                    y2={8}
                    stroke={segmentColors.sub.line}
                    strokeWidth={3}
                  />
                  <circle cx={4} cy={8} r={3} fill={segmentColors.sub.point} />
                  <circle cx={20} cy={8} r={3} fill={segmentColors.sub.point} />
                </svg>
                <span className="font-mono text-[10px] text-zinc-500">
                  SubAgent
                </span>
              </div>
            </div>
            {subAgentCount > 0 && (
              <Badge
                variant="outline"
                className="border-cyan-500/20 bg-cyan-500/5 font-mono text-xs text-cyan-400"
              >
                {subAgentCount} SubAgent{subAgentCount !== 1 ? 's' : ''}
              </Badge>
            )}
          </div>
        </div>
      </CardHeader>

      <CardContent className="relative flex-1 overflow-hidden p-0">
        <div
          ref={containerRef}
          className="max-h-[500px] overflow-auto"
        >
          <svg width={svgWidth} height={svgHeight} className="bg-zinc-900">
            {/* Grid background */}
            <defs>
              <pattern
                id="grid"
                width="40"
                height="40"
                patternUnits="userSpaceOnUse"
              >
                <path
                  d="M 40 0 L 0 0 0 40"
                  fill="none"
                  stroke="#3f3f46"
                  strokeWidth="0.5"
                />
              </pattern>
            </defs>
            <rect width="100%" height="100%" fill="url(#grid)" />

            {/* Lane backgrounds and labels */}
            {segmentData.segments.map((segment) => {
              const y = centerY + segment.yPosition
              return (
                <g key={`lane-${segment.id}`}>
                  {/* Lane background stripe */}
                  <rect
                    x={0}
                    y={y - LANE_HEIGHT / 2}
                    width={svgWidth}
                    height={LANE_HEIGHT}
                    fill={segment.id === 'main' ? '#18181b' : '#1f1f23'}
                    opacity={0.5}
                  />
                  {/* Lane label */}
                  <text
                    x={12}
                    y={y + 4}
                    fill={segment.id === 'main' ? '#a1a1aa' : '#71717a'}
                    fontSize={11}
                    fontFamily="monospace"
                    fontWeight={segment.id === 'main' ? 600 : 400}
                  >
                    {segment.label}
                  </text>
                  {/* Lane centerline (dashed for sub-agents) */}
                  <line
                    x1={80}
                    y1={y}
                    x2={svgWidth - 20}
                    y2={y}
                    stroke={segment.id === 'main' ? '#3f3f46' : '#27272a'}
                    strokeWidth={1}
                    strokeDasharray={segment.id === 'main' ? 'none' : '4,4'}
                  />
                </g>
              )
            })}

            {/* Fork lines connecting Main to SubAgent start points */}
            {segmentData.segments
              .filter((segment) => segment.id !== 'main')
              .map((subSegment) => {
                const x = timestampToX(
                  subSegment.startTime,
                  segmentData.timeRange,
                  svgWidth,
                )
                const mainY = centerY
                const subY = centerY + subSegment.yPosition

                return (
                  <path
                    key={`fork-${subSegment.id}`}
                    d={`M ${x} ${mainY} Q ${x} ${(mainY + subY) / 2} ${x} ${subY}`}
                    fill="none"
                    stroke="#6366f1"
                    strokeWidth={2}
                    opacity={0.6}
                  />
                )
              })}

            {/* Segment lines and endpoints */}
            {segmentData.segments.map((segment) => {
              const startX = timestampToX(
                segment.startTime,
                segmentData.timeRange,
                svgWidth,
              )
              const endX = timestampToX(
                segment.endTime,
                segmentData.timeRange,
                svgWidth,
              )
              const y = centerY + segment.yPosition
              const colors = getSegmentColors(segment)
              const isHovered = hoveredSegment?.id === segment.id
              const isSelected = selectedSegmentId === segment.id

              return (
                <g
                  key={`segment-${segment.id}`}
                  className="cursor-pointer"
                  onMouseEnter={() => setHoveredSegment(segment)}
                  onMouseLeave={() => setHoveredSegment(null)}
                  onClick={() => handleSegmentClick(segment)}
                >
                  {/* Selection highlight */}
                  {isSelected && (
                    <rect
                      x={startX - 8}
                      y={y - 12}
                      width={endX - startX + 16}
                      height={24}
                      rx={4}
                      fill={colors.line}
                      opacity={0.15}
                    />
                  )}

                  {/* Segment line */}
                  <line
                    x1={startX}
                    y1={y}
                    x2={endX}
                    y2={y}
                    stroke={isHovered || isSelected ? colors.hover : colors.line}
                    strokeWidth={
                      isHovered || isSelected
                        ? SEGMENT_LINE_WIDTH + 2
                        : SEGMENT_LINE_WIDTH
                    }
                    strokeLinecap="round"
                    className="transition-all duration-150"
                  />

                  {/* Start point */}
                  <circle
                    cx={startX}
                    cy={y}
                    r={
                      isHovered || isSelected
                        ? ENDPOINT_RADIUS + 2
                        : ENDPOINT_RADIUS
                    }
                    fill={isHovered || isSelected ? colors.hover : colors.point}
                    stroke={colors.line}
                    strokeWidth={2}
                    className="transition-all duration-150"
                  />

                  {/* End point */}
                  <circle
                    cx={endX}
                    cy={y}
                    r={
                      isHovered || isSelected
                        ? ENDPOINT_RADIUS + 2
                        : ENDPOINT_RADIUS
                    }
                    fill={isHovered || isSelected ? colors.hover : colors.point}
                    stroke={colors.line}
                    strokeWidth={2}
                    className="transition-all duration-150"
                  />

                  {/* Message count badge */}
                  {segment.messageCount > 1 && (
                    <g>
                      <rect
                        x={(startX + endX) / 2 - 12}
                        y={y - 22}
                        width={24}
                        height={14}
                        rx={7}
                        fill="#27272a"
                        stroke={colors.line}
                        strokeWidth={1}
                      />
                      <text
                        x={(startX + endX) / 2}
                        y={y - 12}
                        fill="#e4e4e7"
                        fontSize={9}
                        fontFamily="monospace"
                        textAnchor="middle"
                      >
                        {segment.messageCount}
                      </text>
                    </g>
                  )}
                </g>
              )
            })}

            {/* Tooltip for hovered segment */}
            {hoveredSegment && (() => {
              const startX = timestampToX(
                hoveredSegment.startTime,
                segmentData.timeRange,
                svgWidth,
              )
              const endX = timestampToX(
                hoveredSegment.endTime,
                segmentData.timeRange,
                svgWidth,
              )
              const y = centerY + hoveredSegment.yPosition
              const tooltipX = (startX + endX) / 2

              return (
                <g>
                  <rect
                    x={tooltipX - 50}
                    y={y + 14}
                    width={100}
                    height={34}
                    rx={4}
                    fill="#27272a"
                    stroke="#3f3f46"
                    strokeWidth={1}
                  />
                  <text
                    x={tooltipX}
                    y={y + 28}
                    fill="#e4e4e7"
                    fontSize={10}
                    fontFamily="monospace"
                    textAnchor="middle"
                  >
                    {hoveredSegment.label}
                  </text>
                  <text
                    x={tooltipX}
                    y={y + 40}
                    fill="#71717a"
                    fontSize={9}
                    fontFamily="monospace"
                    textAnchor="middle"
                  >
                    {formatTime(Number(hoveredSegment.startTime.seconds))} -{' '}
                    {formatTime(Number(hoveredSegment.endTime.seconds))}
                  </text>
                </g>
              )
            })()}

            {/* Time axis */}
            <g>
              {/* Start time */}
              <text
                x={40}
                y={svgHeight - 12}
                fill="#52525b"
                fontSize={10}
                fontFamily="monospace"
              >
                0:00
              </text>
              {/* End time */}
              {segmentData.totalDuration > 0 && (
                <text
                  x={svgWidth - 60}
                  y={svgHeight - 12}
                  fill="#52525b"
                  fontSize={10}
                  fontFamily="monospace"
                  textAnchor="end"
                >
                  {formatTime(segmentData.timeRange.end)}
                </text>
              )}
            </g>
          </svg>
        </div>
      </CardContent>

      {/* Bottom accent */}
      <div className="absolute bottom-0 left-0 h-[2px] w-0 bg-gradient-to-r from-violet-500 to-cyan-500 transition-all duration-700 group-hover:w-full" />
    </Card>
  )
}
