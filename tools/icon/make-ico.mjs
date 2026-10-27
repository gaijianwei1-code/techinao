// 用 site logo 生成多尺寸 ico（Windows 快捷方式/安装包图标）
import sharp from "sharp";
import { readFileSync, writeFileSync } from "node:fs";

const [, , svgPath, icoPath] = process.argv;
const svg = readFileSync(svgPath);

const sizes = [16, 24, 32, 48, 64, 128, 256];
const pngs = [];
for (const size of sizes) {
    const png = await sharp(svg, { density: 512 }).resize(size, size, { fit: "contain", background: { r: 0, g: 0, b: 0, alpha: 0 } }).png().toBuffer();
    pngs.push({ size, png });
}

// 手写 ICO 容器：ICONDIR + ICONDIRENTRY[] + PNG 数据
const header = Buffer.alloc(6);
header.writeUInt16LE(0, 0);
header.writeUInt16LE(1, 2);
header.writeUInt16LE(pngs.length, 4);

let offset = 6 + pngs.length * 16;
const entries = [];
for (const { size, png } of pngs) {
    const entry = Buffer.alloc(16);
    entry.writeUInt8(size >= 256 ? 0 : size, 0);
    entry.writeUInt8(size >= 256 ? 0 : size, 1);
    entry.writeUInt8(0, 2);
    entry.writeUInt8(0, 3);
    entry.writeUInt16LE(1, 4);
    entry.writeUInt16LE(32, 6);
    entry.writeUInt32LE(png.length, 8);
    entry.writeUInt32LE(offset, 12);
    entries.push(entry);
    offset += png.length;
}

writeFileSync(icoPath, Buffer.concat([header, ...entries, ...pngs.map((p) => p.png)]));
console.log("ico written:", icoPath, sizes.join(","));
