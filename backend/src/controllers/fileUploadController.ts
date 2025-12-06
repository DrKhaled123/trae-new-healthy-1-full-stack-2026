import { Request, Response } from 'express';
import { AppError } from '../middleware/errorHandler';
import { uploadFile, uploadMultipleFiles } from '../services/fileUploadService';

export const uploadAvatar = async (req: Request, res: Response, next: () => void) => {
  try {
    const { file, error } = await uploadFile(req.files?.avatar);
    if (error) {
      throw new AppError(error.message, 400, 'FILE_UPLOAD_ERROR');
    }
    res.status(200).json({ success: true, data: file });
  } catch (err) {
    next(err);
  }
};

export const uploadProgressPhotos = async (req: Request, res: Response, next: () => void) => {
  try {
    const { files, error } = await uploadMultipleFiles(req.files?.photos);
    if (error) {
      throw new AppError(error.message, 400, 'FILE_UPLOAD_ERROR');
    }
    res.status(200).json({ success: true, data: files });
  } catch (err) {
    next(err);
  }
};
