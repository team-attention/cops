import { create } from 'zustand'
import { persist } from 'zustand/middleware'

// UserData represents the authenticated user's information.
interface UserData {
  id: string
  email: string
  name: string
  avatarUrl: string
}

// OrganizationData represents a user's organization membership.
interface OrganizationData {
  id: string
  name: string
  role: 'admin' | 'member'
}

// UserStoreState defines the state shape.
interface UserStoreState {
  user: UserData | null
  organizations: OrganizationData[]
  selectedOrganizationId: string | null
  isLoading: boolean
  error: string | null
}

// UserStoreActions defines the available actions.
interface UserStoreActions {
  setUser: (user: UserData | null) => void
  setOrganizations: (organizations: OrganizationData[]) => void
  setSelectedOrganizationId: (id: string | null) => void
  setLoading: (isLoading: boolean) => void
  setError: (error: string | null) => void
  reset: () => void
}

type UserStore = UserStoreState & UserStoreActions

const initialState: UserStoreState = {
  user: null,
  organizations: [],
  selectedOrganizationId: null,
  isLoading: false,
  error: null,
}

export const useUserStore = create<UserStore>()(
  persist(
    (set) => ({
      ...initialState,

      setUser: (user) => set({ user }),

      setOrganizations: (organizations) =>
        set((state) => ({
          organizations,
          // Auto-select first organization if none selected or current selection invalid
          selectedOrganizationId:
            state.selectedOrganizationId &&
            organizations.some((org) => org.id === state.selectedOrganizationId)
              ? state.selectedOrganizationId
              : organizations.length > 0
                ? organizations[0].id
                : null,
        })),

      setSelectedOrganizationId: (selectedOrganizationId) =>
        set({ selectedOrganizationId }),

      setLoading: (isLoading) => set({ isLoading }),

      setError: (error) => set({ error }),

      reset: () => set(initialState),
    }),
    {
      name: 'cops-user-storage',
      partialize: (state) => ({
        selectedOrganizationId: state.selectedOrganizationId,
      }),
    }
  )
)
