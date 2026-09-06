import { apiClient } from "@/lib/apiClient";

interface WsTicketResponse {
  ticket: string;
  expires_in: number;
}

/**
 * Short-lived ticket authenticating one WebSocket handshake — see
 * docs/adr/0005-websocket-ticket-auth.md. Fetch a fresh one per connection attempt;
 * it expires in seconds, so one held across a reconnect backoff arrives dead.
 */
export async function fetchWsTicket(): Promise<string> {
  const { ticket } = await apiClient<WsTicketResponse>("/ws-ticket");
  return ticket;
}
