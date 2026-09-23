import type { InjectionKey, Ref } from 'vue'

/** Which SpecCard a SpecStack currently holds open, and how to change it. */
export interface SpecFocus {
  /** Null means no card has been clicked, so each card follows its own `active`. */
  focused: Ref<string | null>
  toggle: (name: string) => void
}

export const SpecFocusKey: InjectionKey<SpecFocus> = Symbol('spec-focus')
