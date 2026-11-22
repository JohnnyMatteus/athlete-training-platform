import { sha256 } from '@noble/hashes/sha2.js';
import { bytesToHex } from '@noble/hashes/utils.js';

const SLICE = 8 * 1024 * 1024;

// Hash incremental em fatias: o export do app Saúde pode ter GBs e não cabe em memória.
export async function sha256Hex(blob: Blob, onProgress?: (fraction: number) => void): Promise<string> {
  const h = sha256.create();
  for (let off = 0; off < blob.size; off += SLICE) {
    h.update(new Uint8Array(await blob.slice(off, off + SLICE).arrayBuffer()));
    onProgress?.(Math.min(1, (off + SLICE) / blob.size));
  }
  return bytesToHex(h.digest());
}
