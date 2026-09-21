/** Response of POST /api/auth/demo — a fresh sandbox session. */
export interface DemoSession {
  id: string;
  email: string;
  full_name: string;
  /** The sandbox board seeded for this visitor; where the client should land them. */
  board_id: string;
  /** RFC 3339. After this the sandbox is purged server-side. */
  expires_at: string;
}
