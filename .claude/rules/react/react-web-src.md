---
trigger: always_on
globs: **/src/**/*.{ts,tsx}
paths: **/src/**/*.{ts,tsx}
---

# Source Directory Rules

## Feature Driven Development

This project follows Feature Driven Development (FDD) architecture. All features should be organized under the `feature/` directory with clear separation of concerns.

## Directory Structure

```
src/
├── route/                  # TanStack Router pages (screen components)
├── feature/                # Feature modules
│   └── {feature-name}/
│       ├── component/      # Feature-specific components
│       ├── hook/           # Feature-specific hooks
│       ├── util/           # Feature-specific utilities
│       └── type/           # Feature-specific types
├── shared/                 # Shared resources across features
│   ├── component/          # Shared components
│   ├── hook/               # Shared hooks
│   ├── service/            # Shared services (business logic & external library wrappers)
│   ├── util/               # Shared utilities
│   └── type/               # Shared types
├── gen/                    # Generated code (DO NOT EDIT MANUALLY)
│   ├── grpcstub/           # Generated gRPC/Connect stubs from protobuf
│   └── shadcn/             # shadcn/ui generated components
│       ├── component/      # UI components from shadcn
│       └── lib/            # shadcn utilities
└── asset/                  # Static assets (images, fonts, etc.)
```

## Generated Code Rules

**IMPORTANT: Files under `src/gen/` are programmatically generated and must NOT be edited manually.**

- **`gen/grpcstub/`**: Generated from Protocol Buffers using `buf generate`
  - Contains TypeScript clients for gRPC/Connect services
  - Regenerated whenever proto files change
  - Import from `@/gen/grpcstub/{service}/v1/...`

- **`gen/shadcn/`**: Generated from shadcn/ui CLI
  - Installed via `npx shadcn@latest add <component-name>`
  - Configuration in `components.json`
  - Import from `@/gen/shadcn/component/...`

**To modify generated code:**
- For gRPC stubs: Update `.proto` files in `idl/protobuf/` and run `buf generate`
- For shadcn components: Extend in `shared/component/` or feature-specific directories


## Naming Conventions

### Directory Names
- **All directory names must be in singular form**
  - Correct: `component/`, `hook/`, `util/`, `type/`
  - Incorrect: `components/`, `hooks/`, `utils/`, `types/`
- Use lowercase with hyphens for multi-word feature names
  - Correct: `user-profile/`, `camera-capture/`
  - Incorrect: `userProfile/`, `CameraCapture/`

### File Names
- **All files**: kebab-case (lowercase with hyphens)
- **Component files**: Use `.tsx` extension
  - Correct: `camera-button.tsx`, `user-profile-card.tsx`
  - Incorrect: `CameraButton.tsx`, `UserProfileCard.tsx`
- **Hooks**: Add `use-` prefix with `.ts` extension
  - Correct: `use-camera-permission.ts`, `use-auth.ts`
- **Other files** (services, utilities, types): Use `.ts` extension
  - Correct: `api-client.ts`, `format-date.ts`, `camera-config.ts`

## Feature Organization

Each feature should be self-contained and follow this structure:

```
feature/
└── camera-capture/
    ├── component/
    │   ├── camera-preview.tsx
    │   └── camera-controls.tsx
    ├── hook/
    │   ├── use-camera-device.ts
    │   └── use-camera-capture.ts
    ├── util/
    │   └── process-image.ts
    └── type/
        └── camera-config.ts
```

## Shared Resources

Resources used across multiple features should be placed in the `shared/` directory:

```
shared/
├── component/
│   ├── button.tsx
│   └── loading.tsx
├── hook/
│   └── use-auth.ts
├── service/
│   ├── llm.ts           # LLM service for AI-generated content
│   └── api-client.ts    # API communication service
├── util/
│   └── format-date.ts
└── type/
    ├── llm.ts           # LLM-related types
    └── api-response.ts
```

### UI Components

**Use shadcn/ui as the primary UI component system:**

- **Prefer shadcn/ui components** over building custom UI components from scratch
- Install shadcn components using the CLI: `npx shadcn@latest add <component-name>`
- Only create custom UI components when:
  - The required component doesn't exist in shadcn/ui
  - You need significant customization that extends beyond shadcn's composability
- When customizing shadcn components:
  - Extend existing shadcn components rather than creating from scratch
  - Keep customizations in `shared/component/` or feature-specific `component/` directories
- All shadcn-installed components remain in `src/gen/shadcn/component/`

### Service Layer

Services are used to encapsulate business logic and external library integrations:

- Place services in `shared/service/` for cross-feature usage
- Use singleton pattern for services that maintain state
- Services should handle initialization, error handling, and cleanup
- Export instances rather than classes when appropriate

#### gRPC API Integration Pattern

This project uses **ConnectRPC** with **TanStack Query** for API communication. All gRPC stubs are auto-generated from Protocol Buffers.

**Feature Structure:**

```
feature/
└── {feature-name}/
    └── hook/
        └── use-{rpc-method}.ts     # Query/Mutation hooks wrapping gRPC calls
```

**Hook Naming Convention:**
- Name after the RPC method, not HTTP verbs
- One hook file per RPC method for clarity

**Hook Implementation Pattern:**

```ts
// feature/{feature-name}/hook/use-{method-name}.ts
import { useQuery } from "@connectrpc/connect-query";
import { methodName } from "@/gen/grpcstub/{service}/v1/{service}-{Service}_connectquery";
import { transport } from "@/shared/service/connect-transport";

export const useMethodName = (input?: InputType) => {
  return useQuery(methodName, input, { transport });
};
```

**Real Example (Project):**

```ts
// feature/project/hook/use-get-project.ts
import { useQuery } from "@connectrpc/connect-query";
import { getProject } from "@/gen/grpcstub/project/v1/project-ProjectService_connectquery";
import { transport } from "@/shared/service/connect-transport";

export const useGetProject = (input: { id: string }) => {
  return useQuery(getProject, input, { transport });
};
```

**For Mutations (Write Operations):**

```ts
import { useMutation } from "@connectrpc/connect-query";
import { createUser } from "@/gen/grpcstub/user/v1/user-UserService_connectquery";
import { transport } from "@/shared/service/connect-transport";

export const useCreateUser = () => {
  return useMutation(createUser, { transport });
};
```

**Guidelines:**
- Import generated stubs from `@/gen/grpcstub/{service}/v1/`
- Use shared `transport` from `@/shared/service/connect-transport`
- Pass TanStack Query options (e.g., `refetchInterval`, `enabled`) as needed
- Keep hook logic minimal - just wrap the generated function

## Import Rules

- Feature components should import from their own feature directory first
- Use shared resources for cross-feature functionality
- Avoid circular dependencies between features
- Use absolute imports from `@/` (configured in tsconfig.json)

Example:
```tsx
// Good
import { CameraPreview } from '@/feature/camera-capture/component/camera-preview';
import { Button } from '@/shared/component/button';

// Avoid
import { CameraPreview } from '../../../feature/camera-capture/component/camera-preview';
```