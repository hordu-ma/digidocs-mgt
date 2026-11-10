#!/usr/bin/env python3
"""TCP bridge from the Docker app network to model servers bound on loopback.

The local model servers (llama.cpp Qwen on 127.0.0.1:11500, Ollama on
127.0.0.1:11434) only listen on loopback, which containers cannot reach. This
forwards selected ports on the Docker bridge gateway (172.29.0.1) to them, the
same pattern as the existing openclaw-loopback-proxy for the OpenClaw gateway.

DigiDocs uses it to call the local model directly instead of going through the
personal OpenClaw agent, which injected ~25k prompt tokens per request and
routed document text to a cloud model (see docs/optimization-20261110.md).

Mappings come from DIGIDOCS_LLM_BRIDGE="listen_host:listen_port=target_host:target_port,..."
"""

import asyncio
import os
import signal

DEFAULT_MAPPINGS = "172.29.0.1:11501=127.0.0.1:11500,172.29.0.1:11502=127.0.0.1:11434"


async def pipe(reader: asyncio.StreamReader, writer: asyncio.StreamWriter) -> None:
    try:
        while data := await reader.read(65536):
            writer.write(data)
            await writer.drain()
    except (asyncio.CancelledError, ConnectionError):
        pass
    finally:
        writer.close()
        try:
            await writer.wait_closed()
        except ConnectionError:
            pass


def make_handler(target_host: str, target_port: int):
    async def handle(client_reader: asyncio.StreamReader, client_writer: asyncio.StreamWriter) -> None:
        try:
            target_reader, target_writer = await asyncio.open_connection(target_host, target_port)
        except OSError:
            client_writer.close()
            await client_writer.wait_closed()
            return
        await asyncio.gather(pipe(client_reader, target_writer), pipe(target_reader, client_writer))

    return handle


def parse(spec: str) -> list[tuple[str, int, str, int]]:
    out = []
    for item in filter(None, (s.strip() for s in spec.split(","))):
        listen, target = item.split("=")
        lh, lp = listen.rsplit(":", 1)
        th, tp = target.rsplit(":", 1)
        out.append((lh, int(lp), th, int(tp)))
    return out


async def main() -> None:
    servers = []
    for lh, lp, th, tp in parse(os.environ.get("DIGIDOCS_LLM_BRIDGE", DEFAULT_MAPPINGS)):
        servers.append(await asyncio.start_server(make_handler(th, tp), lh, lp))
        print(f"bridge {lh}:{lp} -> {th}:{tp}", flush=True)
    stop = asyncio.Event()
    loop = asyncio.get_running_loop()
    for sig in (signal.SIGTERM, signal.SIGINT):
        loop.add_signal_handler(sig, stop.set)
    await stop.wait()
    for s in servers:
        s.close()
        await s.wait_closed()


if __name__ == "__main__":
    asyncio.run(main())
