import sharp from 'sharp';
import { mkdirSync } from 'fs';

mkdirSync('public/icons', { recursive: true });

const sizes = [192, 512];
for (const size of sizes) {
  await sharp({
    create: {
      width: size,
      height: size,
      channels: 4,
      background: { r: 26, g: 26, b: 26, alpha: 1 },
    },
  })
    .png()
    .toFile(`public/icons/icon-${size}.png`);
  console.log(`Created public/icons/icon-${size}.png`);
}
