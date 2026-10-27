// 生成两个 ico：
//   logo.ico      —— PNG 压缩条目，用于快捷方式 / 注册表图标
//   installer.ico —— BMP 条目，NSIS 编译器对安装程序图标的兼容性要求
import sharp from "sharp";
import { readFileSync, writeFileSync } from "node:fs";

const [, , svgPath, logoPath, installerPath] = process.argv;
const svg = readFileSync(svgPath);

async function renderPng(size) {
    return sharp(svg, { density: 512 })
        .resize(size, size, { fit: "contain", background: { r: 0, g: 0, b: 0, alpha: 0 } })
        .png()
        .toBuffer();
}

async function renderBmp(size) {
    const { data } = await sharp(svg, { density: 512 })
        .resize(size, size, { fit: "contain", background: { r: 0, g: 0, b: 0, alpha: 0 } })
        .ensureAlpha()
        .raw()
        .toBuffer({ resolveWithObject: true });

    const header = Buffer.alloc(40);
    header.writeUInt32LE(40, 0);
    header.writeInt32LE(size, 4);
    header.writeInt32LE(size * 2, 8);
    header.writeUInt16LE(1, 12);
    header.writeUInt16LE(32, 14);
    header.writeUInt32LE(0, 16);
    header.writeUInt32LE(size * size * 4, 20);

    const pixels = Buffer.alloc(size * size * 4);
    for (let y = 0; y < size; y++) {
        for (let x = 0; x < size; x++) {
            const src = (y * size + x) * 4;
            const dst = ((size - 1 - y) * size + x) * 4;
            pixels[dst] = data[src + 2];
            pixels[dst + 1] = data[src + 1];
            pixels[dst + 2] = data[src];
            pixels[dst + 3] = data[src + 3];
        }
    }

    const maskRow = Math.ceil(size / 8 / 4) * 4;
    const mask = Buffer.alloc(maskRow * size);
    return Buffer.concat([header, pixels, mask]);
}

function buildIco(images) {
    const header = Buffer.alloc(6);
    header.writeUInt16LE(0, 0);
    header.writeUInt16LE(1, 2);
    header.writeUInt16LE(images.length, 4);

    let offset = 6 + images.length * 16;
    const entries = [];
    for (const { size, data } of images) {
        const entry = Buffer.alloc(16);
        entry.writeUInt8(size >= 256 ? 0 : size, 0);
        entry.writeUInt8(size >= 256 ? 0 : size, 1);
        entry.writeUInt8(0, 2);
        entry.writeUInt8(0, 3);
        entry.writeUInt16LE(1, 4);
        entry.writeUInt16LE(32, 6);
        entry.writeUInt32LE(data.length, 8);
        entry.writeUInt32LE(offset, 12);
        entries.push(entry);
        offset += data.length;
    }
    return Buffer.concat([header, ...entries, ...images.map((i) => i.data)]);
}

const sizes = [16, 24, 32, 48, 64, 128, 256];
const pngImages = [];
for (const size of sizes) pngImages.push({ size, data: await renderPng(size) });
writeFileSync(logoPath, buildIco(pngImages));

const bmpImages = [];
for (const size of [16, 32, 48, 64, 128, 256]) bmpImages.push({ size, data: await renderBmp(size) });
writeFileSync(installerPath, buildIco(bmpImages));

console.log("written:", logoPath, installerPath);
