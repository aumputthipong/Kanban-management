export interface DemoSession {
  id: string;
  email: string;
  full_name: string;
  board_id: string;
  /** RFC 3339; the sandbox is purged after this. */
  expires_at: string;
}
