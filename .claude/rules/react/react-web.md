---
trigger: always_on
globs: **/*.{ts,tsx}
paths: **/*.{ts,tsx}
---

# Typescript & React Rules

## Type Rule

### 1. Never use `any` type
- Always use explicit types instead of `any`
- If the type is unknown, use `unknown` and perform type narrowing
- Use generics for flexible yet type-safe code

### 2. Use specific and discriminated union types
Prefer discriminated unions over optional properties for better type safety:

```ts
// Wrong: Ambiguous type with optional properties
export interface WrongType {
  isSuccess: boolean;
  error?: Error;
  message?: string;
}

// Correct: Discriminated union with specific types
interface CorrectSuccessType {
  isSuccess: true;
  message: string;
}

interface CorrectFailureType {
  isSuccess: false;
  error: Error;
}

export type CorrectType = CorrectSuccessType | CorrectFailureType;
```

### 3. Use generics to minimize type code duplication
Create reusable, generalized types when possible:

```ts
// Good: Generic type for API responses
interface ApiSuccessResBody<T> {
  success: true;
  data: T;
}

interface ApiFailureResBody {
  success: false;
  error: Error;
}

type ApiResBody<T> = ApiSuccessResBody<T> | ApiFailureResBody;

// Usage
type UserResBody = ApiResBody<User>;
type PostResBody = ApiResBody<Post>;
```

### 4. Always define named types instead of inline/anonymous types
Define explicit interfaces or types for component props and function parameters:

```ts
// Wrong: Inline/anonymous type
const Component = (props: { onPress: () => void }) => {
  // ...
};

// Correct: Named interface with proper event types
import type { MouseEventHandler } from 'react';

interface ComponentProps {
  onClick: MouseEventHandler<HTMLButtonElement>;
}

export const Component = (props: ComponentProps) => {
  // ...
};
```

### 5. Reuse types from external packages
Always use types provided by packages instead of redefining them:

```ts
// Wrong: Redefining types that already exist in packages
interface MyCompletionParams {
  prompt?: string;
  messages?: Array<{ role: string; content: string }>;
  temperature?: number;
  // ... duplicating package types
}

// Correct: Import and use package types directly
import type { CompletionParams, LlamaContext } from 'llama.rn';

function buildParams(options: MyOptions): CompletionParams {
  return {
    prompt: options.prompt,
    temperature: options.temperature,
    // ... using the actual package type
  };
}
```

## Component Rule
- **Always use named exports** - Never use default exports
- Components must use named export with arrow function:
  ```tsx
  export const ComponentName = () => {
    return <Container>...</Container>;
  };
  ```