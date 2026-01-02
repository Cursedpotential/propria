from .base import BaseExporter
from .txt import TxtExporter
from .json import JsonExporter
from .zip import ZipExporter

EXPORTERS = {
    'TXT': TxtExporter(),
    'JSON': JsonExporter(),
    'ZIP': ZipExporter(),
}
