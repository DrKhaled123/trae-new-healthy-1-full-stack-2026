import { Request } from 'express';
import path from 'path';
import fs from 'fs';
import sharp from 'sharp';

// Save uploaded file
export const saveFile = async (file: Express.Multer.File, folder: string = 'uploads') => {
  const filename = `${Date.now()}-${file.originalname}`;
  const dir = path.join(__dirname, '..', folder);
  // Ensure directory exists
  await fs.promises.mkdir(dir, { recursive: true });
  const filepath = path.join(dir, filename);
  await fs.promises.writeFile(filepath, file.buffer);
  return filepath;
};

// Resize image
export const resizeImage = async (filepath: string, width: number, height?: number) => {
  const outputPath = filepath.replace(/\.(\w+)$/, '-resized.$1');
  await sharp(filepath).resize(width, height).toFile(outputPath);
  return outputPath;
};

// Delete file
export const deleteFile = async (filepath: string) => {
  await fs.promises.unlink(filepath);
};
