import { useMemo } from 'react'
import {  timestampMs } from '@bufbuild/protobuf/wkt'
import type {Timestamp} from '@bufbuild/protobuf/wkt';
import type { FeaturedMember } from '@/gen/grpcstub/dashboard/v1/dashboard_pb'

interface RaceTrackProps {
  members: Array<FeaturedMember>
  sinceUnix: bigint
}

interface CharacterProps {
  idle?: boolean
}

// ---------------------------------------------------------------------------
// Homer Simpson — bald, big belly, white shirt, blue pants
// ---------------------------------------------------------------------------
function HomerCharacter({ idle }: CharacterProps) {
  return (
    <svg width="48" height="52" viewBox="0 0 48 52" fill="none">
      <g className={idle ? '' : 'homer-arm-back'}>
        <path
          d="M18 28 L10 34 L7 38"
          stroke="#E5E7EB"
          strokeWidth="4"
          strokeLinecap="round"
          fill="none"
        />
      </g>
      <g className={idle ? '' : 'homer-leg-back'}>
        <rect x="16" y="38" width="6" height="9" rx="2" fill="#3B82F6" />
        <ellipse cx="19" cy="48" rx="5" ry="2.5" fill="#1C1917" />
      </g>
      <rect x="13" y="35" width="20" height="4" rx="1" fill="#78716C" />
      <ellipse cx="22" cy="29" rx="11" ry="9" fill="#F9FAFB" />
      <path
        d="M16 22 Q22 24 28 22"
        stroke="#E5E7EB"
        strokeWidth="1.5"
        fill="none"
      />
      <ellipse cx="24" cy="32" rx="3" ry="1.5" fill="#E5E7EB" />
      <rect x="13" y="38" width="20" height="8" rx="2" fill="#2563EB" />
      <g className={idle ? '' : 'homer-leg-front'}>
        <rect x="22" y="38" width="6" height="9" rx="2" fill="#1D4ED8" />
        <ellipse cx="26" cy="48" rx="5.5" ry="2.5" fill="#1C1917" />
      </g>
      <g className={idle ? '' : 'homer-arm-front'}>
        <path
          d="M26 26 L34 31 L37 35"
          stroke="#FBBF24"
          strokeWidth="4"
          strokeLinecap="round"
          fill="none"
        />
        <circle cx="37" cy="36" r="3" fill="#FBBF24" />
      </g>
      <rect x="18" y="19" width="8" height="5" rx="2" fill="#FBBF24" />
      <ellipse cx="24" cy="12" rx="12" ry="11" fill="#FBBF24" />
      <ellipse cx="13" cy="13" rx="3" ry="3.5" fill="#FBBF24" />
      <ellipse cx="13" cy="13" rx="1.5" ry="2" fill="#F59E0B" />
      <ellipse cx="27" cy="19" rx="5" ry="2.5" fill="#D1D5DB" opacity="0.5" />
      <ellipse cx="29" cy="17" rx="5" ry="3.5" fill="#FBBF24" />
      <path d="M24 18 Q30 16 34 18 Q32 22 26 21 Z" fill="#F5F5F4" />
      <path
        d="M24 18 Q30 16 34 18"
        stroke="#1C1917"
        strokeWidth="1"
        fill="none"
      />
      <path
        d="M25 20 Q29 22 33 20"
        stroke="#D97706"
        strokeWidth="1"
        fill="none"
      />
      <ellipse cx="33" cy="14" rx="5" ry="4" fill="#FBBF24" />
      <ellipse cx="34" cy="13" rx="4" ry="3.5" fill="#F59E0B" />
      <ellipse cx="35" cy="15" rx="1.5" ry="1" fill="#D97706" />
      <ellipse cx="26" cy="10" rx="4.5" ry="4" fill="white" />
      <circle cx="27.5" cy="10.5" r="2" fill="#1C1917" />
      <circle cx="28.5" cy="9.5" r="0.7" fill="white" />
      <path
        d="M22 8 Q26 6 30 8"
        stroke="#1C1917"
        strokeWidth="1.2"
        fill="none"
      />
      <path
        d="M18 4 Q17 1 19 0"
        stroke="#1C1917"
        strokeWidth="1.5"
        strokeLinecap="round"
        fill="none"
      />
      <path
        d="M21 3 Q20 0 22 -1"
        stroke="#1C1917"
        strokeWidth="1.5"
        strokeLinecap="round"
        fill="none"
      />
      <path
        d="M24 3 Q24 0 26 -1"
        stroke="#1C1917"
        strokeWidth="1.5"
        strokeLinecap="round"
        fill="none"
      />
    </svg>
  )
}

// ---------------------------------------------------------------------------
// Bart Simpson — spiky hair, red t-shirt, blue shorts
// ---------------------------------------------------------------------------
function BartCharacter({ idle }: CharacterProps) {
  return (
    <svg width="48" height="52" viewBox="0 0 48 52" fill="none">
      <polygon
        points="22,18 19,8 23,14 21,4 25,12 24,2 27,11 26,5 30,12 28,7 32,13 31,9 34,15 30,18"
        fill="#FBBF24"
        stroke="#d97706"
        strokeWidth="0.4"
        strokeLinejoin="round"
      />
      <ellipse cx="27" cy="22" rx="9" ry="8" fill="#FBBF24" />
      <path d="M18,23 Q18,30 24,31 Q30,32 36,27 L35,22" fill="#FBBF24" />
      <ellipse
        cx="18"
        cy="23"
        rx="2.2"
        ry="2.5"
        fill="#FBBF24"
        stroke="#d97706"
        strokeWidth="0.4"
      />
      <circle cx="32" cy="21" r="3.2" fill="white" />
      <circle cx="33" cy="21.5" r="1.8" fill="#1a1a1a" />
      <circle cx="33.6" cy="20.8" r="0.6" fill="white" />
      <path
        d="M29,19.5 Q32,17.5 35.2,19.5"
        stroke="#1a1a1a"
        strokeWidth="1.1"
        fill="none"
        strokeLinecap="round"
      />
      <path
        d="M29,17.5 Q32,15.5 35,17"
        stroke="#1a1a1a"
        strokeWidth="1.4"
        fill="none"
        strokeLinecap="round"
      />
      <path
        d="M35,23 Q38,22 37,25 Q36,26 35,25"
        fill="#FBBF24"
        stroke="#d97706"
        strokeWidth="0.5"
      />
      <path
        d="M30,28 Q33,30 35,27"
        stroke="#1a1a1a"
        strokeWidth="1"
        fill="none"
        strokeLinecap="round"
      />
      <rect x="22" y="30" width="7" height="4" fill="#FBBF24" />
      <path
        d="M16,34 Q14,34 14,42 L34,42 Q34,34 32,34 L28,34 Q28,32 25,32 Q22,32 22,34 Z"
        fill="#EF4444"
      />
      <path d="M32,34 Q36,33 37,37 L34,38 Q33,36 32,36 Z" fill="#EF4444" />
      <path d="M16,34 Q12,33 11,37 L14,38 Q15,36 16,36 Z" fill="#EF4444" />
      <g className={idle ? '' : 'bart-arm-back'}>
        <line
          x1="15"
          y1="36"
          x2="10"
          y2="44"
          stroke="#FBBF24"
          strokeWidth="3.5"
          strokeLinecap="round"
        />
        <circle cx="10" cy="44.5" r="2.2" fill="#FBBF24" />
      </g>
      <g className={idle ? '' : 'bart-arm-front'}>
        <line
          x1="33"
          y1="36"
          x2="38"
          y2="44"
          stroke="#FBBF24"
          strokeWidth="3.5"
          strokeLinecap="round"
        />
        <circle cx="38" cy="44.5" r="2.2" fill="#FBBF24" />
      </g>
      <path
        d="M14,42 L14,48 L22,48 L24,43 L26,48 L34,48 L34,42 Z"
        fill="#3B82F6"
      />
      <g className={idle ? '' : 'bart-leg-back'}>
        <line
          x1="19"
          y1="48"
          x2="17"
          y2="52"
          stroke="#FBBF24"
          strokeWidth="4"
          strokeLinecap="round"
        />
        <ellipse cx="15" cy="52" rx="4" ry="2" fill="#3B82F6" />
        <rect x="13" y="50" width="5" height="2.5" fill="#3B82F6" rx="1" />
      </g>
      <g className={idle ? '' : 'bart-leg-front'}>
        <line
          x1="29"
          y1="48"
          x2="31"
          y2="52"
          stroke="#FBBF24"
          strokeWidth="4"
          strokeLinecap="round"
        />
        <ellipse cx="33" cy="52" rx="4" ry="2" fill="#3B82F6" />
        <rect x="30" y="50" width="5" height="2.5" fill="#3B82F6" rx="1" />
      </g>
    </svg>
  )
}

// ---------------------------------------------------------------------------
// Lisa Simpson — starburst hair, pearl necklace, red dress
// ---------------------------------------------------------------------------
function LisaCharacter({ idle }: CharacterProps) {
  return (
    <svg width="48" height="52" viewBox="0 0 48 52" fill="none">
      <polygon
        points="22,0 24,6 26,0 27,6 31,1 29,7 34,3 30,8 36,6 31,10 37,10 31,13 37,15 31,15 36,19 30,17 34,22 28,17 29,23 25,17 24,23 21,17 19,23 18,17 13,22 15,17 10,19 14,15 8,16 14,13 8,10 14,10 9,6 15,8 11,3 16,7 18,1 20,6"
        fill="#FBBF24"
      />
      <ellipse cx="24" cy="12" rx="8" ry="8" fill="#FBBF24" />
      <ellipse cx="25" cy="22" rx="9" ry="8.5" fill="#FBBF24" />
      <ellipse cx="16" cy="23" rx="2" ry="2.5" fill="#F0B920" />
      <ellipse cx="30" cy="20" rx="2.8" ry="3" fill="white" />
      <circle cx="31" cy="20.5" r="1.8" fill="#1E293B" />
      <circle cx="31.5" cy="19.8" r="0.6" fill="white" />
      <path
        d="M27 17 L34 16.5"
        stroke="#D97706"
        strokeWidth="1.4"
        strokeLinecap="round"
      />
      <circle cx="34" cy="22" r="1" fill="#E0A010" />
      <path
        d="M30 26 Q33 28 35 26"
        stroke="#92400E"
        strokeWidth="1.1"
        fill="none"
        strokeLinecap="round"
      />
      <circle
        cx="20"
        cy="31"
        r="1.2"
        fill="white"
        stroke="#D1D5DB"
        strokeWidth="0.3"
      />
      <circle
        cx="22.5"
        cy="30"
        r="1.2"
        fill="white"
        stroke="#D1D5DB"
        strokeWidth="0.3"
      />
      <circle
        cx="25"
        cy="29.5"
        r="1.2"
        fill="white"
        stroke="#D1D5DB"
        strokeWidth="0.3"
      />
      <circle
        cx="27.5"
        cy="30"
        r="1.2"
        fill="white"
        stroke="#D1D5DB"
        strokeWidth="0.3"
      />
      <circle
        cx="30"
        cy="31"
        r="1.2"
        fill="white"
        stroke="#D1D5DB"
        strokeWidth="0.3"
      />
      <g className={idle ? '' : 'lisa-arm-back'}>
        <path
          d="M19 34 L10 38 L8 43"
          stroke="#FBBF24"
          strokeWidth="4"
          strokeLinecap="round"
          fill="none"
        />
        <circle cx="8" cy="44" r="2" fill="#FBBF24" />
      </g>
      <rect x="16" y="32" width="16" height="6" rx="2" fill="#EF4444" />
      <path d="M14 38 L18 50 L32 50 L34 38Z" fill="#EF4444" />
      <path d="M22 38 L20 50 L26 50 L26 38Z" fill="#DC2626" opacity="0.3" />
      <g className={idle ? '' : 'lisa-leg-back'}>
        <rect x="17" y="48" width="5" height="6" rx="2" fill="#FBBF24" />
        <ellipse cx="17" cy="52" rx="4.5" ry="2.2" fill="#DC2626" />
      </g>
      <g className={idle ? '' : 'lisa-leg-front'}>
        <rect x="24" y="48" width="5" height="6" rx="2" fill="#F5C030" />
        <ellipse cx="27" cy="52" rx="4.5" ry="2.2" fill="#EF4444" />
      </g>
      <g className={idle ? '' : 'lisa-arm-front'}>
        <path
          d="M30 34 L38 30 L41 26"
          stroke="#F5C030"
          strokeWidth="4"
          strokeLinecap="round"
          fill="none"
        />
        <circle cx="41" cy="25" r="2" fill="#FBBF24" />
      </g>
    </svg>
  )
}

// ---------------------------------------------------------------------------
// Marge Simpson — tall blue beehive hair, green dress
// ---------------------------------------------------------------------------
function MargeCharacter({ idle }: CharacterProps) {
  return (
    <svg width="48" height="60" viewBox="0 0 48 60" fill="none">
      <ellipse cx="22" cy="22" rx="8" ry="4" fill="#3B82F6" />
      <rect x="14" y="3" width="16" height="20" rx="8" fill="#3B82F6" />
      <rect
        x="16"
        y="4"
        width="5"
        height="16"
        rx="3"
        fill="#60A5FA"
        opacity="0.45"
      />
      <ellipse cx="22" cy="3" rx="8" ry="4" fill="#2563EB" />
      <ellipse cx="24" cy="26" rx="9" ry="9" fill="#FBBF24" />
      <ellipse cx="15" cy="26" rx="2" ry="2.5" fill="#F59E0B" />
      <ellipse cx="29" cy="24" rx="2.8" ry="3" fill="white" />
      <circle cx="30" cy="24.5" r="1.8" fill="#1E293B" />
      <circle cx="30.5" cy="23.8" r="0.6" fill="white" />
      <path
        d="M27 21.5 Q29 20 31 21"
        stroke="#1E293B"
        strokeWidth="1"
        fill="none"
        strokeLinecap="round"
      />
      <path
        d="M33 27 Q36 26 35 29"
        stroke="#F59E0B"
        strokeWidth="1.4"
        fill="none"
        strokeLinecap="round"
      />
      <path
        d="M30 31 Q33 33 35 31"
        stroke="#92400E"
        strokeWidth="1.1"
        fill="none"
        strokeLinecap="round"
      />
      <circle cx="18" cy="35" r="1.4" fill="#EF4444" />
      <circle cx="22" cy="34.2" r="1.4" fill="#EF4444" />
      <circle cx="26" cy="34.5" r="1.4" fill="#EF4444" />
      <circle cx="29" cy="35.5" r="1.4" fill="#EF4444" />
      <path
        d="M14 36 Q14 28 24 28 Q34 28 34 36 L36 56 L12 56 Z"
        fill="#22C55E"
      />
      <path
        d="M16 36 Q17 30 22 29"
        stroke="#4ADE80"
        strokeWidth="1.2"
        fill="none"
        strokeLinecap="round"
        opacity="0.6"
      />
      <g className={idle ? '' : 'marge-arm-back'}>
        <path
          d="M17 37 L10 43 L7 49"
          stroke="#22C55E"
          strokeWidth="4"
          strokeLinecap="round"
          fill="none"
        />
        <circle cx="7" cy="49" r="2.5" fill="#FBBF24" />
      </g>
      <g className={idle ? '' : 'marge-leg-back'}>
        <rect x="14" y="46" width="6" height="8" rx="3" fill="#16A34A" />
        <ellipse cx="14" cy="55" rx="5.5" ry="2.8" fill="#15803D" />
        <ellipse cx="14" cy="54.5" rx="5.5" ry="1.5" fill="#166534" />
      </g>
      <g className={idle ? '' : 'marge-leg-front'}>
        <rect x="22" y="46" width="6" height="8" rx="3" fill="#15803D" />
        <ellipse cx="26" cy="55" rx="5.5" ry="2.8" fill="#166534" />
        <ellipse cx="26" cy="54.5" rx="5.5" ry="1.5" fill="#14532D" />
      </g>
      <g className={idle ? '' : 'marge-arm-front'}>
        <path
          d="M28 37 L36 42 L39 46"
          stroke="#16A34A"
          strokeWidth="4"
          strokeLinecap="round"
          fill="none"
        />
        <circle cx="39" cy="46" r="2.5" fill="#FBBF24" />
      </g>
    </svg>
  )
}

// ---------------------------------------------------------------------------
// Maggie Simpson — blue onesie, pacifier, blue bow
// ---------------------------------------------------------------------------
function MaggieCharacter({ idle }: CharacterProps) {
  return (
    <svg width="40" height="52" viewBox="0 0 40 52" fill="none">
      <g className={idle ? '' : 'maggie-leg-back'}>
        <rect x="10" y="38" width="7" height="6" rx="3" fill="#2563EB" />
        <ellipse cx="13" cy="45" rx="5" ry="2.5" fill="#1D4ED8" />
      </g>
      <ellipse cx="20" cy="34" rx="11" ry="9" fill="#3B82F6" />
      <ellipse cx="21" cy="34" rx="6" ry="5.5" fill="#60A5FA" opacity="0.45" />
      <ellipse cx="20" cy="42" rx="6" ry="3" fill="#3B82F6" />
      <g className={idle ? '' : 'maggie-arm-back'}>
        <path
          d="M12 30 L5 35 L3 38"
          stroke="#3B82F6"
          strokeWidth="5"
          strokeLinecap="round"
          fill="none"
        />
        <circle cx="3" cy="38" r="2.8" fill="#FBBF24" />
      </g>
      <g className={idle ? '' : 'maggie-leg-front'}>
        <rect x="18" y="38" width="7" height="6" rx="3" fill="#2563EB" />
        <ellipse cx="21" cy="45" rx="5" ry="2.5" fill="#1D4ED8" />
      </g>
      <g className={idle ? '' : 'maggie-arm-front'}>
        <path
          d="M26 29 L33 33 L35 36"
          stroke="#2563EB"
          strokeWidth="5"
          strokeLinecap="round"
          fill="none"
        />
        <circle cx="35" cy="36" r="2.8" fill="#FBBF24" />
      </g>
      <circle cx="22" cy="14" r="12" fill="#FBBF24" />
      <ellipse cx="10" cy="14" rx="2.2" ry="2.8" fill="#F5C040" />
      <path
        d="M14 5 L16 0 L18 4 L20 -1 L22 3 L24 -2 L26 2 L28 -1 L29 4 L31 1 L31 6 L14 6Z"
        fill="#FBBF24"
      />
      <path d="M16 4 Q13 0 15 2 Q17 4 20 4Z" fill="#2563EB" />
      <path d="M24 4 Q27 0 25 2 Q23 4 20 4Z" fill="#2563EB" />
      <circle cx="20" cy="4" r="2" fill="#1D4ED8" />
      <ellipse cx="27" cy="13" rx="3.5" ry="4" fill="white" />
      <circle cx="28" cy="13.5" r="2.2" fill="#1E293B" />
      <circle cx="28.8" cy="12.5" r="0.8" fill="white" />
      <path
        d="M24 10 Q26 8.5 29 10"
        stroke="#1E293B"
        strokeWidth="1"
        fill="none"
        strokeLinecap="round"
      />
      <path
        d="M24.5 9 Q27 7.5 30 9"
        stroke="#D97706"
        strokeWidth="1.2"
        fill="none"
        strokeLinecap="round"
      />
      <ellipse cx="33" cy="15" rx="1.8" ry="1.2" fill="#F0A020" />
      <ellipse cx="33" cy="20" rx="3.5" ry="2.5" fill="#F97316" />
      <ellipse cx="36.5" cy="20" rx="1.8" ry="1.2" fill="#EA580C" />
      <circle
        cx="33"
        cy="20"
        r="1.2"
        fill="none"
        stroke="#C2410C"
        strokeWidth="0.8"
      />
    </svg>
  )
}

// ---------------------------------------------------------------------------
// Ned Flanders — brown hair, thick mustache, glasses, green sweater
// ---------------------------------------------------------------------------
function NedCharacter({ idle }: CharacterProps) {
  return (
    <svg width="48" height="52" viewBox="0 0 48 52" fill="none">
      <g className={idle ? '' : 'ned-leg-back'}>
        <rect x="13" y="38" width="5" height="9" rx="2" fill="#7C3AED" />
        <ellipse cx="15" cy="48" rx="5" ry="2.5" fill="#1C1917" />
      </g>
      <rect x="10" y="24" width="20" height="15" rx="3" fill="#22C55E" />
      <path
        d="M16 24 Q20 28 24 24"
        stroke="#16A34A"
        strokeWidth="1.5"
        fill="none"
        strokeLinecap="round"
      />
      <line
        x1="10"
        y1="32"
        x2="30"
        y2="32"
        stroke="#16A34A"
        strokeWidth="0.8"
        opacity="0.5"
      />
      <line
        x1="10"
        y1="36"
        x2="30"
        y2="36"
        stroke="#16A34A"
        strokeWidth="0.8"
        opacity="0.5"
      />
      <g className={idle ? '' : 'ned-leg-front'}>
        <rect x="21" y="38" width="5" height="9" rx="2" fill="#6D28D9" />
        <ellipse cx="24" cy="48" rx="5" ry="2.5" fill="#292524" />
      </g>
      <g className={idle ? '' : 'ned-arm-back'}>
        <rect x="6" y="24" width="5" height="9" rx="2.5" fill="#16A34A" />
        <ellipse cx="8" cy="34" rx="3" ry="2.5" fill="#FBBF24" />
      </g>
      <g className={idle ? '' : 'ned-arm-front'}>
        <rect x="29" y="23" width="5" height="9" rx="2.5" fill="#15803D" />
        <ellipse cx="31" cy="33" rx="3" ry="2.5" fill="#FBBF24" />
      </g>
      <rect x="17" y="19" width="8" height="6" rx="1" fill="#FBBF24" />
      <ellipse cx="22" cy="11" rx="10" ry="10" fill="#FBBF24" />
      <path
        d="M12 8 Q13 1 22 1 Q31 1 32 8 Q30 5 22 5 Q16 5 12 8Z"
        fill="#78350F"
      />
      <path
        d="M16 5 Q19 3.5 22 5"
        stroke="#92400E"
        strokeWidth="0.8"
        fill="none"
        strokeLinecap="round"
      />
      <ellipse cx="12" cy="11" rx="2" ry="2.5" fill="#F5C518" />
      <line
        x1="25"
        y1="11"
        x2="28"
        y2="11"
        stroke="#374151"
        strokeWidth="1"
        strokeLinecap="round"
      />
      <circle
        cx="30"
        cy="11"
        r="3.2"
        stroke="#374151"
        strokeWidth="1.2"
        fill="white"
        fillOpacity="0.15"
      />
      <circle
        cx="23.5"
        cy="11"
        r="2.5"
        stroke="#374151"
        strokeWidth="1"
        fill="white"
        fillOpacity="0.1"
      />
      <line
        x1="12.5"
        y1="11"
        x2="14"
        y2="11"
        stroke="#374151"
        strokeWidth="0.8"
        strokeLinecap="round"
      />
      <circle cx="30" cy="11" r="1.5" fill="#1E293B" />
      <circle cx="30.5" cy="10.5" r="0.5" fill="white" />
      <path
        d="M27 7.5 Q30 6.5 33 7.5"
        stroke="#78350F"
        strokeWidth="1.3"
        fill="none"
        strokeLinecap="round"
      />
      <ellipse cx="34" cy="13" rx="2.8" ry="2.2" fill="#F5A623" />
      <path
        d="M25 17 Q27 15.5 30 16 Q32 15.5 34 16.5 Q37 16 38 17.5 Q37.5 19 35 18 Q32.5 19.5 30 17.5 Q27.5 19.5 25 18 Q23 18.5 23 17 Q24 16 25 17Z"
        fill="#78350F"
      />
      <path
        d="M27 21 Q30 23 34 21"
        stroke="#92400E"
        strokeWidth="1.2"
        fill="none"
        strokeLinecap="round"
      />
      <path
        d="M27.5 21.5 Q30 22.5 33.5 21.5"
        stroke="white"
        strokeWidth="0.7"
        fill="none"
        strokeLinecap="round"
      />
      <ellipse cx="35" cy="17" rx="2" ry="1.2" fill="#FCA5A5" opacity="0.35" />
    </svg>
  )
}

// ---------------------------------------------------------------------------
// Mr. Burns — hunched, hooked nose, dark suit, skeletal
// ---------------------------------------------------------------------------
function BurnsCharacter({ idle }: CharacterProps) {
  return (
    <svg width="48" height="52" viewBox="0 0 48 52" fill="none">
      <g className={idle ? '' : 'burns-leg-back'}>
        <rect x="17" y="38" width="3" height="7" rx="1.5" fill="#374151" />
        <line
          x1="19"
          y1="44"
          x2="22"
          y2="50"
          stroke="#374151"
          strokeWidth="2.5"
          strokeLinecap="round"
        />
        <ellipse cx="23" cy="50.5" rx="3.5" ry="1.5" fill="#0F172A" />
      </g>
      <path
        d="M14 24 Q10 26 11 36 L30 36 L29 24 Q22 22 14 24Z"
        fill="#1E293B"
      />
      <path
        d="M14 24 Q12 18 16 16 Q20 14 22 16"
        fill="none"
        stroke="#1E293B"
        strokeWidth="5"
        strokeLinecap="round"
      />
      <path
        d="M19 24 L21 28 L23 24"
        fill="white"
        stroke="white"
        strokeWidth="0.5"
      />
      <line
        x1="21"
        y1="27"
        x2="21"
        y2="34"
        stroke="#0F172A"
        strokeWidth="1.2"
        strokeLinecap="round"
      />
      <rect
        x="10"
        y="33"
        width="4"
        height="2"
        rx="1"
        fill="white"
        opacity="0.8"
      />
      <g className={idle ? '' : 'burns-leg-front'}>
        <rect x="21" y="37" width="3" height="7" rx="1.5" fill="#0F172A" />
        <line
          x1="23"
          y1="43"
          x2="26"
          y2="50"
          stroke="#0F172A"
          strokeWidth="2.5"
          strokeLinecap="round"
        />
        <ellipse cx="27" cy="50.5" rx="3.5" ry="1.5" fill="#1E293B" />
      </g>
      <g className={idle ? '' : 'burns-arm-back'}>
        <line
          x1="16"
          y1="26"
          x2="10"
          y2="31"
          stroke="#374151"
          strokeWidth="2.2"
          strokeLinecap="round"
        />
        <line
          x1="10"
          y1="31"
          x2="8"
          y2="37"
          stroke="#374151"
          strokeWidth="2"
          strokeLinecap="round"
        />
        <circle cx="8" cy="37.5" r="1.6" fill="#D4A849" />
        <line
          x1="7"
          y1="36.5"
          x2="5.5"
          y2="35"
          stroke="#D4A849"
          strokeWidth="1.1"
          strokeLinecap="round"
        />
      </g>
      <g className={idle ? '' : 'burns-arm-front'}>
        <line
          x1="27"
          y1="26"
          x2="33"
          y2="30"
          stroke="#0F172A"
          strokeWidth="2.2"
          strokeLinecap="round"
        />
        <line
          x1="33"
          y1="30"
          x2="35"
          y2="36"
          stroke="#0F172A"
          strokeWidth="2"
          strokeLinecap="round"
        />
        <circle cx="35" cy="36.5" r="1.6" fill="#D4A849" />
        <line
          x1="36"
          y1="35.5"
          x2="37.5"
          y2="34"
          stroke="#D4A849"
          strokeWidth="1.1"
          strokeLinecap="round"
        />
      </g>
      <rect x="19" y="17" width="4" height="8" rx="2" fill="#D4A849" />
      <line
        x1="20"
        y1="17"
        x2="20"
        y2="24"
        stroke="#B8912A"
        strokeWidth="0.5"
        opacity="0.6"
      />
      <line
        x1="22"
        y1="17"
        x2="22"
        y2="24"
        stroke="#B8912A"
        strokeWidth="0.5"
        opacity="0.6"
      />
      <ellipse cx="25" cy="10" rx="9" ry="10" fill="#D4A849" />
      <path
        d="M16 8 Q14 5 15 2 Q16 1 16 4"
        stroke="#5C4A1E"
        strokeWidth="0.8"
        fill="none"
        strokeLinecap="round"
      />
      <path
        d="M17 7 Q14 3 16 0 Q17 0 17 3"
        stroke="#5C4A1E"
        strokeWidth="0.7"
        fill="none"
        strokeLinecap="round"
      />
      <path
        d="M18 7.5 Q17 4 18 1"
        stroke="#5C4A1E"
        strokeWidth="0.6"
        fill="none"
        strokeLinecap="round"
        opacity="0.7"
      />
      <ellipse cx="16" cy="11" rx="1.8" ry="2.2" fill="#C49838" />
      <ellipse cx="29" cy="10" rx="3.5" ry="3" fill="#C49838" />
      <ellipse cx="30" cy="10" rx="2.8" ry="2.4" fill="#FFF8E7" />
      <circle cx="31" cy="10.5" r="1.4" fill="#1E293B" />
      <circle cx="31.5" cy="10" r="0.5" fill="white" />
      <path
        d="M27.5 8.5 Q30 7.5 33 9"
        stroke="#8B6914"
        strokeWidth="1.3"
        fill="none"
        strokeLinecap="round"
      />
      <path
        d="M27.5 11.5 Q30 12.5 33 11.5"
        stroke="#8B6914"
        strokeWidth="0.8"
        fill="none"
        strokeLinecap="round"
      />
      <path
        d="M27 6.5 Q30 5 33 6"
        stroke="#4A3510"
        strokeWidth="1.4"
        fill="none"
        strokeLinecap="round"
      />
      <path
        d="M29 12 Q34 13 36 16 Q37 19 34 20"
        fill="none"
        stroke="#C49838"
        strokeWidth="3.5"
        strokeLinecap="round"
        strokeLinejoin="round"
      />
      <path
        d="M36 16 Q38.5 17 38 19 Q37 21 34 20"
        fill="#C49838"
        stroke="#B8912A"
        strokeWidth="0.5"
      />
      <circle cx="34.5" cy="19.5" r="0.8" fill="#B8912A" opacity="0.6" />
      <path
        d="M29 20.5 Q31 20 33 20.5 Q34 20.5 34.2 20"
        stroke="#8B5E3C"
        strokeWidth="1.1"
        fill="none"
        strokeLinecap="round"
      />
      <path
        d="M29.5 21.5 Q31.5 22.5 33.5 21.5"
        stroke="#8B5E3C"
        strokeWidth="0.8"
        fill="none"
        strokeLinecap="round"
      />
      <path
        d="M29 21 Q31.5 21.2 34 21"
        stroke="#6B4423"
        strokeWidth="0.9"
        fill="none"
      />
      <path
        d="M24 12 Q23 10 24 8"
        stroke="#B8912A"
        strokeWidth="0.5"
        fill="none"
        opacity="0.5"
      />
      <path
        d="M28 13 Q27 14 28 15"
        stroke="#B8912A"
        strokeWidth="0.5"
        fill="none"
        opacity="0.4"
      />
    </svg>
  )
}

// ---------------------------------------------------------------------------
// Krusty the Clown — white face, green hair, red nose, baggy suit
// ---------------------------------------------------------------------------
function KrustyCharacter({ idle }: CharacterProps) {
  return (
    <svg width="48" height="52" viewBox="0 0 48 52" fill="none">
      <g className={idle ? '' : 'krusty-leg-back'}>
        <ellipse cx="11" cy="48" rx="9" ry="3.5" fill="#DC2626" />
        <rect x="10" y="36" width="4" height="13" rx="2" fill="#3B82F6" />
      </g>
      <g className={idle ? '' : 'krusty-leg-front'}>
        <ellipse cx="22" cy="48" rx="9" ry="3.5" fill="#DC2626" />
        <rect x="20" y="36" width="4" height="13" rx="2" fill="#3B82F6" />
      </g>
      <ellipse cx="20" cy="30" rx="11" ry="10" fill="#84CC16" />
      <ellipse cx="20" cy="21" rx="7" ry="4" fill="#FBBF24" />
      <circle cx="20" cy="30" r="1.5" fill="#65A30D" />
      <circle cx="20" cy="35" r="1.5" fill="#65A30D" />
      <g className={idle ? '' : 'krusty-arm-back'}>
        <line
          x1="11"
          y1="24"
          x2="4"
          y2="33"
          stroke="#84CC16"
          strokeWidth="5"
          strokeLinecap="round"
        />
        <circle cx="4" cy="34" r="3.5" fill="#F9FAFB" />
      </g>
      <g className={idle ? '' : 'krusty-arm-front'}>
        <line
          x1="26"
          y1="23"
          x2="35"
          y2="30"
          stroke="#84CC16"
          strokeWidth="5"
          strokeLinecap="round"
        />
        <circle cx="36" cy="30" r="3.5" fill="#F9FAFB" />
      </g>
      <rect x="17" y="17" width="6" height="5" rx="2" fill="#F5F5F5" />
      <ellipse cx="24" cy="12" rx="11" ry="10" fill="#F5F5F5" />
      <circle cx="13" cy="9" r="5.5" fill="#10B981" />
      <circle cx="11" cy="14" r="4" fill="#10B981" />
      <circle cx="14" cy="4" r="4.5" fill="#10B981" />
      <circle cx="17" cy="3" r="3" fill="#10B981" />
      <ellipse cx="24" cy="4" rx="6" ry="3" fill="#F5F5F5" />
      <circle cx="33" cy="10" r="3.5" fill="#10B981" />
      <circle cx="34" cy="14" r="3" fill="#10B981" />
      <rect x="18" y="1" width="10" height="4" rx="1" fill="#1E293B" />
      <rect x="16" y="4" width="14" height="2" rx="1" fill="#1E293B" />
      <rect x="18" y="2" width="10" height="1.5" rx="0.5" fill="#EF4444" />
      <circle cx="30" cy="11" r="1.8" fill="white" />
      <circle cx="30.5" cy="11" r="1" fill="#111827" />
      <path
        d="M28.5 9 Q30.5 8 32.5 9.5"
        stroke="#374151"
        strokeWidth="1"
        fill="none"
        strokeLinecap="round"
      />
      <circle cx="34" cy="14" r="3.5" fill="#EF4444" />
      <circle cx="33" cy="13" r="1" fill="#FCA5A5" opacity="0.6" />
      <path
        d="M31 17 Q34 21 31 20"
        stroke="#DC2626"
        strokeWidth="2.5"
        fill="#DC2626"
        strokeLinecap="round"
      />
      <path
        d="M28 17 Q30 15.5 33 17"
        stroke="#DC2626"
        strokeWidth="1.5"
        fill="#DC2626"
        strokeLinecap="round"
      />
      <circle cx="28" cy="19" r="0.6" fill="#9CA3AF" />
      <circle cx="30" cy="20" r="0.6" fill="#9CA3AF" />
    </svg>
  )
}

// ---------------------------------------------------------------------------
// Ralph Wiggum — big round head, messy brown hair, vacant expression
// ---------------------------------------------------------------------------
function RalphCharacter({ idle }: CharacterProps) {
  return (
    <svg width="48" height="52" viewBox="0 0 48 52" fill="none">
      <g className={idle ? '' : 'ralph-arm-back'}>
        <line
          x1="22"
          y1="28"
          x2="16"
          y2="34"
          stroke="#3B82F6"
          strokeWidth="4"
          strokeLinecap="round"
        />
        <line
          x1="16"
          y1="34"
          x2="13"
          y2="40"
          stroke="#FBBF24"
          strokeWidth="3"
          strokeLinecap="round"
        />
      </g>
      <g className={idle ? '' : 'ralph-leg-back'}>
        <line
          x1="24"
          y1="40"
          x2="18"
          y2="46"
          stroke="#3B82F6"
          strokeWidth="5"
          strokeLinecap="round"
        />
        <line
          x1="18"
          y1="46"
          x2="14"
          y2="52"
          stroke="#FBBF24"
          strokeWidth="4"
          strokeLinecap="round"
        />
        <ellipse cx="12" cy="52" rx="5" ry="2.5" fill="#7C3AED" />
      </g>
      <rect x="18" y="26" width="16" height="16" rx="4" fill="#3B82F6" />
      <path
        d="M22 26 Q24 29 26 26"
        stroke="#2563EB"
        strokeWidth="1.5"
        fill="none"
        strokeLinecap="round"
      />
      <ellipse cx="26" cy="35" rx="8" ry="7" fill="#3B82F6" />
      <rect x="19" y="38" width="14" height="8" rx="2" fill="#2563EB" />
      <rect x="19" y="38" width="14" height="2.5" rx="1" fill="#1D4ED8" />
      <g className={idle ? '' : 'ralph-leg-front'}>
        <line
          x1="27"
          y1="42"
          x2="32"
          y2="47"
          stroke="#3B82F6"
          strokeWidth="5"
          strokeLinecap="round"
        />
        <line
          x1="32"
          y1="47"
          x2="36"
          y2="52"
          stroke="#FBBF24"
          strokeWidth="4"
          strokeLinecap="round"
        />
        <ellipse cx="38" cy="52" rx="5" ry="2.5" fill="#7C3AED" />
      </g>
      <g className={idle ? '' : 'ralph-arm-front'}>
        <line
          x1="30"
          y1="28"
          x2="36"
          y2="33"
          stroke="#3B82F6"
          strokeWidth="4"
          strokeLinecap="round"
        />
        <line
          x1="36"
          y1="33"
          x2="40"
          y2="38"
          stroke="#FBBF24"
          strokeWidth="3"
          strokeLinecap="round"
        />
        <circle cx="40" cy="39" r="2.5" fill="#FBBF24" />
      </g>
      <rect x="22" y="21" width="8" height="7" rx="2" fill="#FBBF24" />
      <ellipse cx="26" cy="13" rx="13" ry="13" fill="#FBBF24" />
      <ellipse cx="38" cy="14" rx="2.5" ry="2" fill="#FBBF24" />
      <ellipse cx="22" cy="4" rx="11" ry="6" fill="#D97706" />
      <ellipse cx="14" cy="7" rx="4" ry="5" fill="#D97706" />
      <ellipse cx="18" cy="2" rx="4" ry="3.5" fill="#B45309" />
      <ellipse cx="25" cy="1" rx="3.5" ry="3" fill="#D97706" />
      <ellipse cx="31" cy="5" rx="3" ry="3.5" fill="#B45309" />
      <ellipse cx="12" cy="10" rx="3" ry="2.5" fill="#D97706" />
      <ellipse cx="33" cy="13" rx="4" ry="4" fill="white" />
      <circle cx="34" cy="13" r="2.5" fill="#1E3A5F" />
      <circle cx="34.5" cy="13" r="1.2" fill="black" />
      <circle cx="35" cy="12" r="0.7" fill="white" />
      <path
        d="M29.5 10.5 Q33 9 36.5 10.5"
        stroke="#D97706"
        strokeWidth="2"
        fill="none"
        strokeLinecap="round"
      />
      <ellipse cx="38" cy="17" rx="3.5" ry="2.5" fill="#F59E0B" />
      <circle cx="40" cy="16.5" r="1" fill="#FCD34D" />
      <ellipse cx="39.5" cy="18" rx="1" ry="0.7" fill="#D97706" />
      <path
        d="M33 20 Q36 23 38.5 20"
        stroke="#8B4513"
        strokeWidth="1.5"
        fill="none"
        strokeLinecap="round"
      />
      <path d="M33.5 20.5 Q36 22.5 38 20.5" fill="white" stroke="none" />
      <ellipse cx="36" cy="22" rx="1.5" ry="0.8" fill="#EF4444" />
      <ellipse cx="36" cy="17" rx="3" ry="2" fill="#FCD34D" opacity="0.4" />
      <rect x="18" y="26" width="4" height="3" rx="1" fill="#60A5FA" />
      <rect x="30" y="26" width="4" height="3" rx="1" fill="#60A5FA" />
    </svg>
  )
}

// ---------------------------------------------------------------------------
// Milhouse — blue hair, big glasses, pink shirt
// ---------------------------------------------------------------------------
function MilhouseCharacter({ idle }: CharacterProps) {
  return (
    <svg width="48" height="52" viewBox="0 0 48 52" fill="none">
      <g className={idle ? '' : 'milhouse-leg-back'}>
        <rect x="19" y="36" width="5" height="8" rx="2" fill="#64748B" />
        <rect x="18" y="43" width="4" height="6" rx="1" fill="#FBBF24" />
        <ellipse cx="19" cy="49.5" rx="5" ry="2.5" fill="#3B82F6" />
        <rect x="14" y="47" width="10" height="3" rx="1.5" fill="#3B82F6" />
        <rect x="13" y="48" width="4" height="2" rx="1" fill="#1D4ED8" />
      </g>
      <g className={idle ? '' : 'milhouse-arm-back'}>
        <rect x="20" y="22" width="4" height="7" rx="2" fill="#F87171" />
        <rect x="21" y="28" width="3.5" height="6" rx="1.5" fill="#FBBF24" />
        <ellipse cx="22.5" cy="35" rx="2.5" ry="2" fill="#FBBF24" />
      </g>
      <rect x="17" y="22" width="13" height="14" rx="3" fill="#F87171" />
      <path
        d="M22 22 Q24 20 26 22"
        fill="white"
        stroke="white"
        strokeWidth="0.5"
      />
      <rect x="17" y="34" width="13" height="6" rx="2" fill="#64748B" />
      <g className={idle ? '' : 'milhouse-leg-front'}>
        <rect x="23" y="36" width="5" height="8" rx="2" fill="#64748B" />
        <rect x="23" y="43" width="4" height="6" rx="1" fill="#FBBF24" />
        <rect x="22" y="47" width="10" height="3" rx="1.5" fill="#3B82F6" />
        <ellipse cx="27" cy="49.5" rx="5" ry="2.5" fill="#3B82F6" />
        <rect x="22" y="48" width="4" height="2" rx="1" fill="#1D4ED8" />
      </g>
      <g className={idle ? '' : 'milhouse-arm-front'}>
        <rect x="27" y="21" width="4" height="7" rx="2" fill="#F87171" />
        <rect x="27.5" y="27" width="3.5" height="6" rx="1.5" fill="#FBBF24" />
        <ellipse cx="29" cy="34" rx="2.5" ry="2" fill="#FBBF24" />
      </g>
      <rect x="22" y="18" width="5" height="5" rx="1" fill="#FBBF24" />
      <ellipse cx="26" cy="13" rx="9" ry="10" fill="#FBBF24" />
      <ellipse cx="17.5" cy="13" rx="2" ry="2.5" fill="#FBBF24" />
      <ellipse cx="17.5" cy="13" rx="1" ry="1.5" fill="#F59E0B" />
      <ellipse cx="26" cy="5" rx="8" ry="5" fill="#3B82F6" />
      <path d="M31 4 Q36 2 35 6 Q33 5 31 6Z" fill="#3B82F6" />
      <path d="M18 6 Q16 4 17 8 Q18 7 19 8Z" fill="#2563EB" />
      <path
        d="M21 4 Q25 2 28 4"
        stroke="#60A5FA"
        strokeWidth="1"
        fill="none"
        strokeLinecap="round"
      />
      <path
        d="M24 3 Q26 0 28 3"
        fill="#3B82F6"
        stroke="#3B82F6"
        strokeWidth="0.5"
      />
      <ellipse cx="34" cy="14" rx="3.5" ry="2.5" fill="#F59E0B" />
      <circle cx="35" cy="13.5" r="1" fill="#FBBF24" />
      <line
        x1="24"
        y1="12"
        x2="27"
        y2="12"
        stroke="#64748B"
        strokeWidth="1.2"
        strokeLinecap="round"
      />
      <circle
        cx="22"
        cy="12"
        r="4"
        fill="none"
        stroke="#64748B"
        strokeWidth="1.5"
      />
      <circle cx="22" cy="12" r="3.5" fill="white" fillOpacity="0.25" />
      <circle
        cx="29"
        cy="12"
        r="4.5"
        fill="none"
        stroke="#64748B"
        strokeWidth="1.8"
      />
      <circle cx="29" cy="12" r="4" fill="white" fillOpacity="0.25" />
      <ellipse
        cx="27.5"
        cy="10.5"
        rx="1.2"
        ry="0.7"
        fill="white"
        fillOpacity="0.6"
        transform="rotate(-25 27.5 10.5)"
      />
      <ellipse
        cx="20.8"
        cy="10.8"
        rx="0.9"
        ry="0.5"
        fill="white"
        fillOpacity="0.5"
        transform="rotate(-20 20.8 10.8)"
      />
      <line
        x1="18"
        y1="12"
        x2="16"
        y2="13"
        stroke="#64748B"
        strokeWidth="1.2"
        strokeLinecap="round"
      />
      <ellipse cx="22" cy="12.5" rx="2" ry="2" fill="white" />
      <ellipse cx="29" cy="12.5" rx="2.5" ry="2" fill="white" />
      <circle cx="23" cy="12.5" r="1.1" fill="#1E293B" />
      <circle cx="30" cy="12.5" r="1.2" fill="#1E293B" />
      <circle cx="23.4" cy="12.1" r="0.35" fill="white" />
      <circle cx="30.4" cy="12.1" r="0.4" fill="white" />
      <path
        d="M20 8.5 Q22 7.5 24 8"
        stroke="#1E293B"
        strokeWidth="1"
        fill="none"
        strokeLinecap="round"
      />
      <path
        d="M27 8 Q29 7 31 7.5"
        stroke="#1E293B"
        strokeWidth="1"
        fill="none"
        strokeLinecap="round"
      />
      <path
        d="M27 17 Q29 16.5 31 17"
        stroke="#C0392B"
        strokeWidth="0.9"
        fill="none"
        strokeLinecap="round"
      />
      <path
        d="M28 17.8 Q29 18.2 30.5 17.8"
        stroke="#F59E0B"
        strokeWidth="0.6"
        fill="none"
        strokeLinecap="round"
      />
      <path
        d="M28 19 Q29 19.5 30 19"
        stroke="#F59E0B"
        strokeWidth="0.5"
        fill="none"
        strokeLinecap="round"
      />
      <path
        d="M22 23 L24 25 L26 23"
        fill="none"
        stroke="#EF4444"
        strokeWidth="0.8"
      />
    </svg>
  )
}

// ---------------------------------------------------------------------------
// Character roster — 10 Simpsons characters, assigned per member index
// ---------------------------------------------------------------------------
const CHARACTERS: Array<React.FC<CharacterProps>> = [
  HomerCharacter,
  BartCharacter,
  LisaCharacter,
  MargeCharacter,
  MaggieCharacter,
  NedCharacter,
  BurnsCharacter,
  KrustyCharacter,
  RalphCharacter,
  MilhouseCharacter,
]

function getCharacter(index: number): React.FC<CharacterProps> {
  return CHARACTERS[index % CHARACTERS.length]
}

// Lane colors for differentiation (10 colors to match 10 characters)
const LANE_COLORS = [
  {
    track: 'from-yellow-500/20 to-yellow-500/5',
    dot: 'bg-yellow-400',
    text: 'text-yellow-400',
  }, // Homer
  {
    track: 'from-red-500/20 to-red-500/5',
    dot: 'bg-red-400',
    text: 'text-red-400',
  }, // Bart
  {
    track: 'from-amber-500/20 to-amber-500/5',
    dot: 'bg-amber-400',
    text: 'text-amber-400',
  }, // Lisa
  {
    track: 'from-blue-500/20 to-blue-500/5',
    dot: 'bg-blue-400',
    text: 'text-blue-400',
  }, // Marge
  {
    track: 'from-sky-500/20 to-sky-500/5',
    dot: 'bg-sky-400',
    text: 'text-sky-400',
  }, // Maggie
  {
    track: 'from-emerald-500/20 to-emerald-500/5',
    dot: 'bg-emerald-400',
    text: 'text-emerald-400',
  }, // Ned
  {
    track: 'from-slate-500/20 to-slate-500/5',
    dot: 'bg-slate-400',
    text: 'text-slate-400',
  }, // Burns
  {
    track: 'from-lime-500/20 to-lime-500/5',
    dot: 'bg-lime-400',
    text: 'text-lime-400',
  }, // Krusty
  {
    track: 'from-orange-500/20 to-orange-500/5',
    dot: 'bg-orange-400',
    text: 'text-orange-400',
  }, // Ralph
  {
    track: 'from-violet-500/20 to-violet-500/5',
    dot: 'bg-violet-400',
    text: 'text-violet-400',
  }, // Milhouse
]

function isCollapsed(lastActivityAt: Timestamp | undefined): boolean {
  if (!lastActivityAt) return true
  const ms = timestampMs(lastActivityAt)
  return Date.now() - ms > 3 * 60 * 1000
}

function getProgressPercent(
  lastActivityAt: FeaturedMember['lastActivityAt'],
  sinceMs: number,
  endMs: number,
): number {
  if (!lastActivityAt) return 0
  const activityMs = timestampMs(lastActivityAt)
  if (activityMs <= sinceMs) return 0
  if (activityMs >= endMs) return 100
  return ((activityMs - sinceMs) / (endMs - sinceMs)) * 100
}

export function RaceTrack({ members, sinceUnix }: RaceTrackProps) {
  const sinceMs = Number(sinceUnix) * 1000
  // Track end = next day 08:00
  const sinceDate = new Date(sinceMs)
  const nextDay8AM = new Date(
    sinceDate.getFullYear(),
    sinceDate.getMonth(),
    sinceDate.getDate() + 1,
    8,
    0,
    0,
    0,
  )
  const endMs = nextDay8AM.getTime()
  const totalMs = endMs - sinceMs

  // Time markers for the track header (every 2 hours)
  const timeMarkers = useMemo(() => {
    const totalHours = totalMs / (60 * 60 * 1000)
    const markers: Array<{ label: string; percent: number }> = []
    for (let h = 0; h <= totalHours; h += 2) {
      const t = new Date(sinceMs + h * 60 * 60 * 1000)
      const label = t.toLocaleTimeString([], {
        hour: '2-digit',
        minute: '2-digit',
        hour12: false,
      })
      markers.push({ label, percent: (h / totalHours) * 100 })
    }
    // Always include end marker (08:00)
    if (markers[markers.length - 1]?.percent !== 100) {
      markers.push({
        label: nextDay8AM.toLocaleTimeString([], {
          hour: '2-digit',
          minute: '2-digit',
          hour12: false,
        }),
        percent: 100,
      })
    }
    return markers
  }, [sinceMs, totalMs, nextDay8AM])

  // Sort members by progress (leader first)
  const sorted = useMemo(() => {
    return [...members].sort((a, b) => {
      const pA = getProgressPercent(a.lastActivityAt, sinceMs, endMs)
      const pB = getProgressPercent(b.lastActivityAt, sinceMs, endMs)
      return pB - pA
    })
  }, [members, sinceMs, endMs])

  return (
    <div className="space-y-2">
      {/* Track time header */}
      <div className="ml-[140px] mr-4 flex justify-between sm:ml-[180px]">
        {timeMarkers.map((m) => (
          <span
            key={m.percent}
            className="font-mono text-[9px] text-zinc-600"
            style={{ marginLeft: m.percent === 0 ? 0 : undefined }}
          >
            {m.label}
          </span>
        ))}
      </div>

      {/* Lanes */}
      {sorted.map((member, idx) => {
        const color = LANE_COLORS[idx % LANE_COLORS.length]
        const progress = getProgressPercent(
          member.lastActivityAt,
          sinceMs,
          endMs,
        )
        const collapsed = isCollapsed(member.lastActivityAt)
        const idle = collapsed || progress === 0 || !member.lastActivityAt
        const Character = getCharacter(idx)

        return (
          <div key={member.userId} className="flex items-center gap-3">
            {/* Member name plate */}
            <div className="w-[128px] shrink-0 sm:w-[168px]">
              <div className="flex items-center gap-2">
                <div className={`h-2 w-2 shrink-0 rounded-full ${color.dot}`} />
                <span className="truncate font-mono text-xs font-bold text-zinc-200">
                  {member.userName || member.userId}
                </span>
              </div>
              <p className="ml-4 truncate font-mono text-[9px] text-zinc-600">
                {member.projectName || '...'}
              </p>
            </div>

            {/* Track lane */}
            <div className="relative flex-1 h-11 rounded-lg border border-zinc-800/60 bg-zinc-900/40 overflow-hidden">
              {/* Lane background gradient */}
              <div
                className={`absolute inset-0 bg-gradient-to-r ${color.track} opacity-40`}
                style={{ width: `${Math.max(progress, 2)}%` }}
              />

              {/* Dashed lane markers */}
              <div className="absolute inset-0 flex items-center">
                <div
                  className="h-[1px] w-full"
                  style={{
                    backgroundImage:
                      'repeating-linear-gradient(to right, rgb(63 63 70 / 0.4) 0px, rgb(63 63 70 / 0.4) 8px, transparent 8px, transparent 20px)',
                  }}
                />
              </div>

              {/* Finish flag */}
              <div className="absolute right-2 top-1/2 -translate-y-1/2 font-mono text-lg opacity-20">
                🏁
              </div>

              {/* Character — runs across the lane, collapses when inactive 3min+ */}
              <div
                className={`absolute top-1/2 transition-all duration-[2000ms] ease-out ${
                  collapsed
                    ? '-translate-y-[30%] opacity-40'
                    : '-translate-y-1/2'
                }`}
                style={{
                  left: `${Math.min(progress, 94)}%`,
                  transform: collapsed
                    ? `translateY(-30%) rotate(90deg)`
                    : undefined,
                  filter: collapsed ? 'grayscale(0.6)' : undefined,
                }}
              >
                <Character idle={idle} />
              </div>

              {/* Stats overlay */}
              <div className="absolute bottom-1 right-2 flex items-center gap-3">
                <span className="font-mono text-[9px] text-zinc-600">
                  {member.promptCount > 0 && `${member.promptCount} prompts`}
                </span>
              </div>
            </div>
          </div>
        )
      })}
    </div>
  )
}
