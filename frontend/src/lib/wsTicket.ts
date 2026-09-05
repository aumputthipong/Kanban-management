import { apiClient } from "@/lib/apiClient";

interface WsTicketResponse {
  ticket: string;
  expires_in: number;
}

/**
 * Fetches a short-lived ticket that authenticates a single WebSocket
 * handshake. Browsers cannot put a header on a WebSocket, and the auth cookie
 * is first-party to the frontend origin, so the socket URL carries this
 * instead — see docs/adr/0005-websocket-ticket-auth.md.
 *
 * The ticket expires in seconds: fetch a fresh one per connection attempt
 * rather than holding one across a reconnect backoff.
 */
export async function fetchWsTicket(): Promise<string> {
  const { ticket } = await apiClient<WsTicketResponse>("/ws-ticket");
  return ticket;
}
