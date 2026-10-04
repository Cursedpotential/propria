"""Screenshot, photo or scanned page: the image-kind classifier. Pure function, no I/O, no model.

> _Byline: Claude Code · Sonnet 5.5 · 2026-10-03_

One job: given facts already read from an image (file name, pixel size, format, whether camera EXIF
exists, whether the
image is a rendered PDF page) say which of three kinds it is and WHY. The reason (``basis``) is
stored beside the kind
so a reviewer can see a wrong call and the rule that made it. Nothing here looks at pixels or calls
a model; a better
classifier replaces this function without touching the stage that calls it.

Kinds: ``screenshot`` (screen capture), ``photo`` (camera picture), ``scan`` (a PDF page rendered to
an image).
The owner's search filter ("only screenshots") and the MaxSim policy
(``INTAKE_IMAGES_MAXSIM=screenshots``) read it.
"""

from __future__ import annotations

from dataclasses import dataclass

KIND_SCREENSHOT = "screenshot"
KIND_PHOTO = "photo"
KIND_SCAN = "scan"
KINDS = (KIND_SCREENSHOT, KIND_PHOTO, KIND_SCAN)

NAME_HINTS = (
    "screenshot",
    "screen shot",
    "screen_shot",
    "screen-shot",
    "screencap",
    "screen_cap",
    "scrnshot",
)
# Native widths of common phone/tablet/desktop screens; a capture is exactly one of these wide
# (portrait or landscape).
SCREEN_WIDTHS = frozenset(
    {
        320,
        360,
        375,
        390,
        393,
        412,
        414,
        428,
        430,
        480,
        540,
        720,
        750,
        768,
        828,
        834,
        1024,
        1080,
        1125,
        1170,
        1179,
        1242,
        1284,
        1290,
        1366,
        1440,
        1536,
        1600,
        1920,
        2048,
        2160,
        2560,
        2732,
        2880,
        3840,
    }
)


@dataclass(frozen=True)
class ImageKind:
    kind: str
    basis: str


def classify_image(
    name: str,
    width: int,
    height: int,
    *,
    format_name: str = "",
    has_camera_exif: bool = False,
    is_pdf_page: bool = False,
    user_comment: str = "",
) -> ImageKind:
    """Classify one image. Rules run in order and the first that fires decides.

    1. a rendered PDF page is a ``scan``;
    2. a file name or EXIF user comment that says screenshot is a ``screenshot`` (``name_hint``);
    3. camera EXIF (make/model/exposure) means a ``photo`` (``camera_exif``);
    4. a PNG without camera EXIF is a ``screenshot`` (``png_no_camera``): cameras write JPEG/HEIC,
    screens write PNG;
    5. a JPEG/WebP without camera EXIF whose width is a native screen width and whose long side is
    at least 1.6 times
       the short side is a ``screenshot`` (``screen_geometry``);
    6. everything else is a ``photo`` (``default``). The basis says it was a default, not a finding.
    """
    lowered = f"{name} {user_comment}".casefold()
    fmt = format_name.upper()
    if is_pdf_page:
        return ImageKind(KIND_SCAN, "pdf_page")
    if any(hint in lowered for hint in NAME_HINTS):
        return ImageKind(KIND_SCREENSHOT, "name_hint")
    if has_camera_exif:
        return ImageKind(KIND_PHOTO, "camera_exif")
    if fmt == "PNG":
        return ImageKind(KIND_SCREENSHOT, "png_no_camera")
    short, long = sorted((width, height))
    if short in SCREEN_WIDTHS and short > 0 and long / short >= 1.6:
        return ImageKind(KIND_SCREENSHOT, "screen_geometry")
    return ImageKind(KIND_PHOTO, "default")
