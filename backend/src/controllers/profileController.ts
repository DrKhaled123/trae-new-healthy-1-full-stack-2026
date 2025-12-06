import { Request, Response } from 'express';
import { AppError } from '../middleware/errorHandler';
import { getUser } from '../middleware/auth';
import profileService from '../services/profileService';

export const getUserProfile = async (req: Request, res: Response, next: () => void) => {
  try {
    const userId = getUser(req).id;
    const profile = await profileService.getProfile(userId);
    res.json({ success: true, data: profile });
  } catch (err) {
    next(err);
  }
};

export const updateUserProfile = async (req: Request, res: Response, next: () => void) => {
  try {
    const userId = getUser(req).id;
    const data = req.body;
    const updatedProfile = await profileService.updateProfile(userId, data);
    res.json({ success: true, data: updatedProfile });
  } catch (err) {
    next(err);
  }
};
