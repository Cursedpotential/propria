// Ported from modules/forks/sbv/frontend/src/utils/vcfParser.js
// MIT, Copyright (c) 2025 lowcarbdev
// Adapted: converted to TypeScript with explicit types; logic and behavior
// (line unfolding, vCard 2.1/3.0/4.0 property/param parsing, quoted-printable
// decode, base64 PHOTO data URIs, address/birthday formatting) is otherwise a
// direct line-for-line port.
// Byline: Claude Code · Opus 5 · 2026-09-20
export interface VCardPhone {
  type: string;
  number: string;
}

export interface VCardEmail {
  type: string;
  address: string;
}

export interface VCardAddress {
  type: string;
  street?: string;
  city?: string;
  state?: string;
  zip?: string;
  country?: string;
}

export interface VCardContact {
  version: string;
  name: string;
  formattedName: string;
  phoneNumbers: VCardPhone[];
  emails: VCardEmail[];
  addresses: VCardAddress[];
  organization: string;
  title: string;
  photo: string | null;
  birthday: string;
  url: string;
  note: string;
}

type VCardParams = Record<string, string>;

export function parseVCard(vcfText: string): VCardContact {
  const contact: VCardContact = {
    version: "",
    name: "",
    formattedName: "",
    phoneNumbers: [],
    emails: [],
    addresses: [],
    organization: "",
    title: "",
    photo: null,
    birthday: "",
    url: "",
    note: "",
  };

  const lines = unfoldLines(vcfText);

  for (const line of lines) {
    const [property, value] = parseVCardLine(line);
    if (!property || value === null) continue;

    const { name, params } = parseProperty(property);

    switch (name.toUpperCase()) {
      case "VERSION":
        contact.version = value;
        break;
      case "FN":
        contact.formattedName = decodeValue(value, params);
        break;
      case "N": {
        const nameParts = value.split(";").map((p) => decodeValue(p, params));
        if (!contact.name) {
          contact.name = [nameParts[3], nameParts[1], nameParts[2], nameParts[0], nameParts[4]]
            .filter((p) => p)
            .join(" ");
        }
        break;
      }
      case "TEL":
        contact.phoneNumbers.push({ type: getTypeLabel(params, "phone"), number: value });
        break;
      case "EMAIL":
        contact.emails.push({ type: getTypeLabel(params, "email"), address: value });
        break;
      case "ADR": {
        const adrParts = value.split(";").map((p) => decodeValue(p, params));
        contact.addresses.push({
          type: getTypeLabel(params, "address"),
          street: adrParts[2],
          city: adrParts[3],
          state: adrParts[4],
          zip: adrParts[5],
          country: adrParts[6],
        });
        break;
      }
      case "ORG":
        contact.organization = decodeValue(value, params);
        break;
      case "TITLE":
        contact.title = decodeValue(value, params);
        break;
      case "PHOTO":
        contact.photo = parsePhoto(value, params);
        break;
      case "BDAY":
        contact.birthday = value;
        break;
      case "URL":
        contact.url = value;
        break;
      case "NOTE":
        contact.note = decodeValue(value, params);
        break;
      default:
        break;
    }
  }

  if (!contact.name && contact.formattedName) contact.name = contact.formattedName;

  return contact;
}

function unfoldLines(text: string): string[] {
  const lines = text.split(/\r?\n/);
  const unfolded: string[] = [];
  let current = "";

  for (const line of lines) {
    if (line.startsWith(" ") || line.startsWith("\t")) {
      current += line.substring(1);
    } else {
      if (current) unfolded.push(current);
      current = line;
    }
  }
  if (current) unfolded.push(current);

  return unfolded;
}

function parseVCardLine(line: string): [string | null, string | null] {
  const colonIndex = line.indexOf(":");
  if (colonIndex === -1) return [null, null];
  return [line.substring(0, colonIndex), line.substring(colonIndex + 1)];
}

function parseProperty(property: string): { name: string; params: VCardParams } {
  const parts = property.split(";");
  const name = parts[0];
  const params: VCardParams = {};

  for (let i = 1; i < parts.length; i++) {
    const param = parts[i];
    const eqIndex = param.indexOf("=");
    if (eqIndex === -1) {
      params.TYPE = params.TYPE ? `${params.TYPE},${param}` : param;
    } else {
      const paramName = param.substring(0, eqIndex);
      const paramValue = param.substring(eqIndex + 1).replace(/^"(.*)"$/, "$1");
      params[paramName.toUpperCase()] = paramValue;
    }
  }

  return { name, params };
}

function getTypeLabel(params: VCardParams, context: "phone" | "email" | "address"): string {
  if (!params.TYPE) {
    return context === "phone" ? "Phone" : context === "email" ? "Email" : "Address";
  }

  const types = params.TYPE.split(",").map((t) => t.toUpperCase());
  const typeMap: Record<string, string> = {
    CELL: "Mobile",
    HOME: "Home",
    WORK: "Work",
    VOICE: "Phone",
    FAX: "Fax",
    PAGER: "Pager",
    MSG: "Message",
    PREF: "Preferred",
    INTERNET: "Email",
  };

  const labels = types
    .map((t) => typeMap[t] || t.charAt(0) + t.substring(1).toLowerCase())
    .filter((l) => l !== "Internet");

  return labels.join(", ") || (context === "phone" ? "Phone" : context === "email" ? "Email" : "Address");
}

function decodeValue(value: string, params: VCardParams): string {
  if (!params.ENCODING) return value;
  if (params.ENCODING.toUpperCase() === "QUOTED-PRINTABLE") return decodeQuotedPrintable(value);
  return value;
}

function decodeQuotedPrintable(str: string): string {
  return str
    .replace(/=\r?\n/g, "")
    .replace(/=([0-9A-F]{2})/gi, (_match, hex: string) => String.fromCharCode(parseInt(hex, 16)));
}

function parsePhoto(value: string, params: VCardParams): string | null {
  const encoding = params.ENCODING ? params.ENCODING.toUpperCase() : "";
  const type = params.TYPE || params.MEDIATYPE || "JPEG";

  if (encoding === "BASE64" || encoding === "B") {
    const base64Data = value.replace(/\s/g, "");
    let mimeType = "image/jpeg";
    const typeUpper = type.toUpperCase();
    if (typeUpper.includes("PNG")) mimeType = "image/png";
    else if (typeUpper.includes("GIF")) mimeType = "image/gif";
    else if (typeUpper.includes("BMP")) mimeType = "image/bmp";
    return `data:${mimeType};base64,${base64Data}`;
  }

  if (value.startsWith("http://") || value.startsWith("https://")) return value;

  return null;
}

export function formatAddress(address: VCardAddress): string {
  const parts = [
    address.street,
    address.city,
    address.state && address.zip ? `${address.state} ${address.zip}` : address.state || address.zip,
    address.country,
  ].filter((p) => p);

  return parts.join(", ");
}

export function formatBirthday(birthday: string): string {
  if (!birthday) return "";

  if (birthday.startsWith("--")) {
    const month = birthday.substring(2, 4);
    const day = birthday.substring(4, 6);
    return `${month}/${day}`;
  }

  if (birthday.includes("-")) {
    const [year, month, day] = birthday.split("-");
    return `${month}/${day}/${year}`;
  }

  if (birthday.length === 8) {
    const year = birthday.substring(0, 4);
    const month = birthday.substring(4, 6);
    const day = birthday.substring(6, 8);
    return `${month}/${day}/${year}`;
  }

  return birthday;
}
