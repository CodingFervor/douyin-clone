import { ref } from 'vue'

// ===================== Shared unread-notification count =====================
// A module-level reactive ref so the Messages view (which owns the data) and
// the App tab bar (which shows the badge) can stay in sync without props or a
// state library. Importers read `unreadTotal`; the Messages view sets it.
export const unreadTotal = ref(0)

export function setUnreadTotal(n) {
  unreadTotal.value = n || 0
}
