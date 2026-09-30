// Byline: Codex · 2026-09-12; write operations + resilient listing added by Claude Code · Sonnet 5 · 2026-09-14.
// Fixed-upstream OpenList adapter for the existing Xplorer UI. Owner direction 2026-09-14: this is
// pre-evidence-space file management -- explicit user/agent-directed move, rename, copy, delete and
// mkdir are in scope (never automatic; dedupe/similarity never deletes on its own).
const UPSTREAM = "https://files.tilapia-skilift.ts.net";
export class StorageError extends Error {
  constructor(message, status = 502) {
    super(message);
    this.status = status;
  }
}
export function storagePath(value = "/") {
  // Only reject what is actually unsafe here: backslash (ambiguous separator), control characters,
  // and "." / ".." traversal segments. Real filenames legitimately contain '%', '?', '#' (seen in this
  // corpus, e.g. a leading '#' and a '%'-bearing .webloc name) -- those are not forbidden, and this
  // path never gets concatenated into a URL without its own encodeURIComponent step (see `asset`).
  if (
    typeof value !== "string" ||
    value.length > 2048 ||
    !value.startsWith("/") ||
    /[\\\x00-\x1f]/.test(value) ||
    value.split("/").some((x) => x === "." || x === "..")
  )
    throw new StorageError("Invalid storage path", 400);
  return value.replace(/\/+$/, "") || "/";
}
const childPath = (path, name) => {
  if (typeof name !== "string" || name.includes("/") || name.includes("\\"))
    throw new StorageError("Invalid provider filename");
  return storagePath(`${path === "/" ? "" : path}/${name}`);
};
const splitParent = (path) => {
  const idx = path.lastIndexOf("/");
  return idx <= 0
    ? { dir: "/", name: path.slice(idx + 1) }
    : { dir: path.slice(0, idx), name: path.slice(idx + 1) };
};
const unix = (value) => {
  const ms = Date.parse(value);
  return Number.isFinite(ms) ? Math.floor(ms / 1000) : 0;
};
const mime = (name) =>
  ({
    txt: "text/plain",
    md: "text/markdown",
    json: "application/json",
    pdf: "application/pdf",
    png: "image/png",
    jpg: "image/jpeg",
    jpeg: "image/jpeg",
    gif: "image/gif",
    webp: "image/webp",
    mp3: "audio/mpeg",
    wav: "audio/wav",
    mp4: "video/mp4",
  })[String(name).split(".").pop().toLowerCase()] || "application/octet-stream";
function entry(item, path) {
  return {
    name: item.name,
    path,
    is_dir: Boolean(item.is_dir),
    size: Number(item.size) || 0,
    modified: unix(item.modified),
    file_type: item.is_dir ? "directory" : item.name.split(".").pop() || "file",
    // Write ops (rename/move/copy/remove/mkdir) are wired below -- entries are no longer
    // reported as read-only. Any specific op OpenList itself refuses still surfaces its own error.
    is_readonly: false,
    mime_type: item.is_dir ? "inode/directory" : mime(item.name),
  };
}
async function limitedBody(response, max) {
  if (Number(response.headers.get("content-length")) > max) {
    await response.body?.cancel();
    throw new StorageError("Selected file exceeds preview limit", 413);
  }
  const chunks = [];
  let total = 0;
  for await (const chunk of response.body) {
    total += chunk.length;
    if (total > max)
      throw new StorageError("Selected file exceeds preview limit", 413);
    chunks.push(chunk);
  }
  return Buffer.concat(chunks);
}
export function createOpenListBridge({
  token,
  fetchImpl = fetch,
  timeoutMs = 15000,
} = {}) {
  async function api(action, args) {
    if (!token)
      throw new StorageError(
        "OpenList server credential is not configured",
        503,
      );
    let response;
    try {
      response = await fetchImpl(`${UPSTREAM}/api/fs/${action}`, {
        method: "POST",
        headers: { authorization: token, "content-type": "application/json" },
        body: JSON.stringify(args),
        redirect: "error",
        signal: AbortSignal.timeout(timeoutMs),
      });
    } catch {
      throw new StorageError("OpenList is unreachable");
    }
    let body;
    try {
      body = JSON.parse(
        (await limitedBody(response, 4 * 1024 * 1024)).toString(),
      );
    } catch (error) {
      if (error instanceof StorageError) throw error;
      throw new StorageError("OpenList returned an invalid response");
    }
    if (!response.ok || body.code !== 200)
      throw new StorageError(
        body.code === 401 || body.code === 403
          ? "OpenList authentication or path permission denied"
          : body.code === 404
            ? "Storage path not found"
            : "Provider listing is unavailable",
        body.code === 404
          ? 404
          : body.code === 401 || body.code === 403
            ? 403
            : 502,
      );
    return body.data;
  }
  async function list(path) {
    path = storagePath(path);
    // 2026-09-17: page through large folders instead of refusing them at 500 entries.
    const MAX_ENTRIES = 20000;
    const data = await api("list", {
      path,
      password: "",
      page: 1,
      per_page: 500,
      refresh: false,
    });
    if (data.total > MAX_ENTRIES)
      throw new StorageError(
        `This folder has ${data.total} entries (browser limit ${MAX_ENTRIES}); use a narrower directory`,
        413,
      );
    data.content = data.content || [];
    for (let page = 2; data.content.length < data.total && page <= MAX_ENTRIES / 500; page++) {
      const more = await api("list", {
        path,
        password: "",
        page,
        per_page: 500,
        refresh: false,
      });
      if (!more.content || more.content.length === 0) break;
      data.content.push(...more.content);
    }
    // A single unrepresentable name (e.g. a real file here starting with '#', or containing raw
    // control characters) must not fail the whole directory -- skip just that entry and keep going.
    const out = [];
    let omitted = 0;
    for (const item of data.content || []) {
      try {
        out.push(entry(item, childPath(path, item.name)));
      } catch {
        omitted++;
      }
    }
    if (omitted > 0)
      console.warn(
        `[openlist-bridge] list(${path}): omitted ${omitted} entr${omitted === 1 ? "y" : "ies"} with an unrepresentable name`,
      );
    return out;
  }
  async function stat(path) {
    path = storagePath(path);
    if (path === "/")
      return {
        name: "OpenList",
        path: "/",
        is_dir: true,
        size: 0,
        modified: 0,
        file_type: "directory",
        is_readonly: true,
      };
    return entry(await api("get", { path, password: "" }), path);
  }
  async function asset(path, range) {
    path = storagePath(path);
    if (path === "/") throw new StorageError("Choose a file", 400);
    // Fetch the raw `get` response ourselves (not just via stat()) because this deployment's
    // /p preview proxy requires the per-file `sign` value as a query param -- found live
    // 2026-09-14: without it, files.tilapia-skilift.ts.net returns "401 expire missing" even
    // with a valid Authorization header. raw_url (the actual signed B2/S3 URL) is still never used.
    const raw = await api("get", { path, password: "" });
    const info = entry(raw, path);
    if (info.is_dir) throw new StorageError("Choose a file", 400);
    const maximum = 8 * 1024 * 1024;
    if (info.size > maximum && !range)
      throw new StorageError(
        "Preview is limited to 8 MiB; select a bounded range",
        413,
      );
    if (range && !/^bytes=\d+-\d+$/.test(range))
      throw new StorageError("A bounded byte range is required", 400);
    if (range) {
      const [start, end] = range.slice(6).split("-").map(Number);
      if (
        !Number.isSafeInteger(start) ||
        !Number.isSafeInteger(end) ||
        end < start ||
        end - start + 1 > maximum
      )
        throw new StorageError("Preview range exceeds limit", 413);
    }
    // /p (WebProxy) returns 403 "proxy not allowed" for this storage -- found live 2026-09-14 --
    // so this uses /d (direct download), which OpenList answers with a redirect to a short-lived
    // signed provider URL. That redirect is followed HERE, server-side, and only the resulting
    // bytes are ever returned to the browser: the signed URL and provider credentials never reach
    // the client. The `sign` query param authenticates this proxy front door, not the provider.
    const encoded = path.split("/").map(encodeURIComponent).join("/");
    const signQuery = raw.sign ? `?sign=${encodeURIComponent(raw.sign)}` : "";
    let response;
    try {
      response = await fetchImpl(`${UPSTREAM}/d${encoded}${signQuery}`, {
        headers: { authorization: token, ...(range ? { range } : {}) },
        redirect: "follow",
        signal: AbortSignal.timeout(timeoutMs),
      });
    } catch {
      throw new StorageError("Selected file preview is unreachable");
    }
    if (!response.ok)
      throw new StorageError(
        "Selected file preview is unavailable",
        response.status === 404 ? 404 : 502,
      );
    const bytes = await limitedBody(response, maximum);
    return {
      bytes,
      status: response.status,
      contentType:
        response.headers.get("content-type") || "application/octet-stream",
      contentRange: response.headers.get("content-range"),
    };
  }
  // ── Write operations (owner direction 2026-09-14: explicit user/agent-directed move, rename,
  // copy, delete and mkdir are in scope pre-evidence-space; never automatic). Each maps onto
  // OpenList's own fs endpoints; OpenList's own errors (permission, unsupported storage) surface
  // through `api()`'s existing mapping rather than being swallowed.
  async function mkdir(path) {
    path = storagePath(path);
    await api("mkdir", { path });
  }
  async function renameInPlace(path, newName) {
    path = storagePath(path);
    if (
      typeof newName !== "string" ||
      newName.includes("/") ||
      newName.includes("\\") ||
      newName === "" ||
      newName === "." ||
      newName === ".."
    )
      throw new StorageError("Invalid new name", 400);
    await api("rename", { path, name: newName });
  }
  async function remove(dir, names) {
    dir = storagePath(dir);
    await api("remove", { dir, names });
  }
  async function moveOrCopy(action, source, destination) {
    const src = splitParent(storagePath(source));
    const dstPath = storagePath(destination);
    // `destination` may be a target directory, or a full target path (same name or a rename).
    // Treat it as a directory if it names an existing directory; otherwise split it as dir+name.
    let dstDir, dstName;
    try {
      const maybeDir = await stat(dstPath);
      if (maybeDir.is_dir) {
        dstDir = dstPath;
        dstName = src.name;
      } else {
        throw new StorageError("Destination file already exists", 409);
      }
    } catch (error) {
      if (error instanceof StorageError && error.status === 409) throw error;
      // Not an existing directory (likely 404) -- treat destination as a full target path.
      const split = splitParent(dstPath);
      dstDir = split.dir;
      dstName = split.name;
    }
    await api(action, { src_dir: src.dir, dst_dir: dstDir, names: [src.name] });
    if (dstName !== src.name) {
      await renameInPlace(childPath(dstDir, src.name), dstName);
    }
  }
  async function command(name, args = {}) {
    const path = () => storagePath(args.path ?? args.filePath);
    switch (name) {
      case "create_dir_recursive":
        return mkdir(path());
      case "rename": {
        const oldPath = storagePath(args.oldPath);
        const { name: newName } = splitParent(storagePath(args.newPath));
        return renameInPlace(oldPath, newName);
      }
      // move_to_trash is deliberately NOT mapped here: OpenList/B2 has no trash/undo, so aliasing
      // the app's "trash" action (which reads as recoverable) to a real remove would be dishonest.
      // remove_file/remove_dir are the explicit, unambiguous delete path.
      case "remove_file":
      case "remove_dir": {
        const { dir, name: fileName } = splitParent(path());
        return remove(dir, [fileName]);
      }
      case "move_file":
        return moveOrCopy("move", args.source, args.destination);
      case "copy":
        return moveOrCopy("copy", args.source, args.destination);
      case "get_user_directories":
        return {
          home: "/",
          documents: "/",
          downloads: "/",
          desktop: "/",
          pictures: "/",
          videos: "/",
          music: "/",
        };
      case "list_drives":
        return (await list("/"))
          .filter((x) => x.is_dir)
          .map((x) => ({
            letter: x.name,
            label: x.name,
            path: x.path,
            total_space: null,
            free_space: null,
          }));
      case "read_directory":
        return list(path());
      case "file_exist":
        try {
          await stat(path());
          return true;
        } catch (error) {
          if (error.status === 404) return false;
          throw error;
        }
      case "is_dir":
        return (await stat(path())).is_dir;
      case "get_file_properties":
      case "get_file_meta_data":
      case "get_detailed_file_properties": {
        const item = await stat(path());
        return {
          ...item,
          file_name: item.name,
          file_path: item.path,
          file_size: item.size,
          created: null,
          accessed: null,
          permissions: "read-only",
          extension: item.is_dir ? "" : item.name.split(".").pop(),
          storage_source: "OpenList",
          capacity_known: false,
        };
      }
      case "read_text_file":
        return (await asset(path())).bytes.toString("utf8");
      case "read_binary_file":
        return [...(await asset(path())).bytes];
      default:
        throw new StorageError(
          `Command is not available in read-only storage mode: ${String(name).slice(0, 80)}`,
          501,
        );
    }
  }
  return { command, asset, locations: () => list("/") };
}
