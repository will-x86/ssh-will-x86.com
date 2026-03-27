export interface Env {
  MESSAGES: KVNamespace;
  WORKER_SECRET: string;
}

interface Message {
  from: string;
  content: string;
  timestamp: string;
}

function unauthorized(msg = "unauthorized"): Response {
  return new Response(msg, { status: 401 });
}

function authenticated(request: Request, env: Env): boolean {
  const secret = new URL(request.url).searchParams.get("secret");
  return secret === env.WORKER_SECRET;
}

// POST /message?secret=...
// Body: { from: string, content: string, timestamp: string }
// Writes to:
//   log:<timestamp>-<uuid>  →  JSON message  (permanent)
//   queue                   →  JSON array of log keys (print queue)
async function handlePost(request: Request, env: Env): Promise<Response> {
  let body: Message;
  try {
    body = await request.json<Message>();
  } catch {
    return new Response("invalid json", { status: 400 });
  }

  if (!body.from || !body.content || !body.timestamp) {
    return new Response("missing fields", { status: 400 });
  }

  const id = crypto.randomUUID();
  const key = `log:${body.timestamp}-${id}`;

  // Write perm log entry
  await env.MESSAGES.put(key, JSON.stringify(body));

  // Append key to queue list
  const queueRaw = await env.MESSAGES.get("queue");
  const queue: string[] = queueRaw ? JSON.parse(queueRaw) : [];
  queue.push(key);
  await env.MESSAGES.put("queue", JSON.stringify(queue));

  return new Response(JSON.stringify({ id: key }), {
    status: 200,
    headers: { "Content-Type": "application/json" },
  });
}

// GET /next?secret=...
// Pops oldest message from queue
//   <from>---<content>---<timestamp>
// Returns 204 if queue is empty.
async function handleNext(env: Env): Promise<Response> {
  const queueRaw = await env.MESSAGES.get("queue");
  const queue: string[] = queueRaw ? JSON.parse(queueRaw) : [];

  if (queue.length === 0) {
    return new Response(null, { status: 204 });
  }

  const key = queue.shift()!;

  // Remove from queue (log entry stays forever)
  await env.MESSAGES.put("queue", JSON.stringify(queue));

  const raw = await env.MESSAGES.get(key);
  if (!raw) {
    // Log entry missing (shouldn't happen)
    return new Response(null, { status: 204 });
  }

  const msg: Message = JSON.parse(raw);
  const body = `${msg.from}---${msg.content}---${msg.timestamp}`;

  return new Response(body, {
    status: 200,
    headers: { "Content-Type": "text/plain" },
  });
}

export default {
  async fetch(request: Request, env: Env): Promise<Response> {
    if (!authenticated(request, env)) {
      return unauthorized();
    }

    const url = new URL(request.url);

    if (request.method === "POST" && url.pathname === "/message") {
      return handlePost(request, env);
    }

    if (request.method === "GET" && url.pathname === "/next") {
      return handleNext(env);
    }

    return new Response("not found", { status: 404 });
  },
};
