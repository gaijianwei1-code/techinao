import sharp from "sharp";
import { readFileSync, writeFileSync } from "node:fs";

const [, , icoPath, pngPath] = process.argv;
const buf = readFileSync(icoPath);
const count = buf.readUInt16LE(4);
const target = 256;
let chosen = null;
for (let i = 0; i < count; i++) {
    const entry = 6 + i * 16;
    const width = buf.readUInt8(entry) || 256;
    const length = buf.readUInt32LE(entry + 8);
    const offset = buf.readUInt32LE(entry + 12);
    if (width === target) chosen = buf.subarray(offset, offset + length);
}
const png = await sharp(chosen).resize(256, 256).png().toBuffer();
writeFileSync(pngPath, png);
console.log("preview written", pngPath);
