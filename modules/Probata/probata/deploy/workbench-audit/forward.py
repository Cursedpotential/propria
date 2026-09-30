"""Relay one TCP port for the length of an audit run (see audit.sh).

Byline: Claude Code · Opus 5.5 · 2026-09-27

    python3 forward.py <listen-host> <listen-port> <target-host> <target-port>

Runs in the foreground on ovh-files for one audit and is stopped when the audit ends. It relays
the devbox network's gateway address to the audit's SSH reverse tunnel on the host loopback, so
the devbox's headless Chrome can reach it. It is not a service.
"""

import asyncio
import sys


async def pipe(reader: asyncio.StreamReader, writer: asyncio.StreamWriter) -> None:
    try:
        while data := await reader.read(65536):
            writer.write(data)
            await writer.drain()
    except (ConnectionError, asyncio.CancelledError):
        pass
    finally:
        writer.close()


async def main(listen_host: str, listen_port: int, target_host: str, target_port: int) -> None:
    async def handle(client_reader: asyncio.StreamReader, client_writer: asyncio.StreamWriter) -> None:
        try:
            upstream_reader, upstream_writer = await asyncio.open_connection(target_host, target_port)
        except OSError:
            client_writer.close()
            return
        await asyncio.gather(pipe(client_reader, upstream_writer), pipe(upstream_reader, client_writer))

    server = await asyncio.start_server(handle, listen_host, listen_port)
    async with server:
        await server.serve_forever()


if __name__ == "__main__":
    asyncio.run(main(sys.argv[1], int(sys.argv[2]), sys.argv[3], int(sys.argv[4])))
