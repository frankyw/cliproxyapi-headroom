"""Create one native CLIProxyAPI plugin archive for the current runner."""
from pathlib import Path
import hashlib
import os
import platform
import re
import zipfile

root = Path(__file__).resolve().parents[1]
version = re.search(r'const version = "([0-9.]+)"', (root / 'plugin.go').read_text()).group(1)
goos = os.environ.get('GOOS') or {'Linux': 'linux', 'Darwin': 'darwin', 'Windows': 'windows'}[platform.system()]
goarch = os.environ.get('GOARCH') or {'x86_64': 'amd64', 'aarch64': 'arm64', 'arm64': 'arm64', 'AMD64': 'amd64'}[platform.machine()]
extension = {'linux': 'so', 'darwin': 'dylib', 'windows': 'dll'}[goos]
library = root / 'dist' / f'headroom.{extension}'
archive = root / 'dist' / f'headroom_{version}_{goos}_{goarch}.zip'
assert library.is_file(), library
with zipfile.ZipFile(archive, 'w', compression=zipfile.ZIP_DEFLATED) as z:
    z.write(library, library.name)
with zipfile.ZipFile(archive) as z:
    assert z.namelist() == [library.name]
    assert z.testzip() is None
if os.environ.get('GITHUB_ACTIONS') != 'true':
    (root / 'dist' / 'checksums.txt').write_text(hashlib.sha256(archive.read_bytes()).hexdigest() + '  ' + archive.name + '\n')
print(archive.name)
