"""Create and validate the official CLIProxyAPI plugin store release layout."""
from pathlib import Path
import hashlib, re, zipfile
root=Path(__file__).resolve().parents[1]
version=re.search(r'const version = "([0-9.]+)"',(root/'plugin.go').read_text()).group(1)
archive=root/'dist'/f'headroom_{version}_linux_amd64.zip'
with zipfile.ZipFile(archive,'w',compression=zipfile.ZIP_DEFLATED) as z:z.write(root/'dist/headroom.so','headroom.so')
with zipfile.ZipFile(archive) as z:assert z.namelist()==['headroom.so'];assert z.testzip() is None
(root/'dist/checksums.txt').write_text(hashlib.sha256(archive.read_bytes()).hexdigest()+'  '+archive.name+'\n')
print(archive.name)
