import Joi from 'joi';

export const updateUserProfileSchema = Joi.object({
  firstName: Joi.string().min(1).max(50).optional(),
  lastName: Joi.string().min(1).max(50).optional(),
  dateOfBirth: Joi.date().iso().optional(),
  gender: Joi.string().valid('male', 'female', 'other').optional(),
  height: Joi.number().positive().max(300).optional(),
  weight: Joi.number().positive().max(1000).optional(),
  activityLevel: Joi.string().valid('sedentary', 'light', 'moderate', 'active', 'very_active').optional(),
  goal: Joi.string().valid('lose_weight', 'maintain', 'gain_weight', 'build_muscle').optional(),
  targetWeight: Joi.number().positive().max(1000).optional(),
  dietaryRestrictions: Joi.array().items(Joi.string()).optional(),
  allergies: Joi.array().items(Joi.string()).optional(),
  units: Joi.string().valid('metric', 'imperial').optional()
});

export const userSettingsSchema = Joi.object({
  notifications: Joi.object({
    email: Joi.boolean().optional(),
    push: Joi.boolean().optional(),
    mealReminders: Joi.boolean().optional(),
    waterReminders: Joi.boolean().optional(),
    goalReminders: Joi.boolean().optional(),
    weeklyReports: Joi.boolean().optional()
  }).optional(),
  privacy: Joi.object({
    profileVisibility: Joi.string().valid('public', 'private', 'friends').optional(),
    shareProgress: Joi.boolean().optional(),
    shareGoals: Joi.boolean().optional()
  }).optional(),
  preferences: Joi.object({
    language: Joi.string().optional(),
    timezone: Joi.string().optional(),
    theme: Joi.string().valid('light', 'dark', 'auto').optional(),
    defaultMealSize: Joi.string().valid('small', 'medium', 'large').optional()
  }).optional()
});

export const changePasswordSchema = Joi.object({
  currentPassword: Joi.string().required().messages({
    'any.required': 'Current password is required'
  }),
  newPassword: Joi.string().min(8).required().messages({
    'string.min': 'Password must be at least 8 characters long',
    'any.required': 'New password is required'
  }),
  confirmPassword: Joi.string().valid(Joi.ref('newPassword')).required().messages({
    'any.only': 'Passwords do not match',
    'any.required': 'Password confirmation is required'
  })
});

export const deleteUserAccountSchema = Joi.object({
  password: Joi.string().required().messages({
    'any.required': 'Password is required to delete account'
  }),
  confirmation: Joi.string().valid('DELETE').required().messages({
    'any.only': 'You must type "DELETE" to confirm account deletion',
    'any.required': 'Confirmation is required'
  })
});

export const updateWeightSchema = Joi.object({
  weight: Joi.number().positive().max(1000).required().messages({
    'number.positive': 'Weight must be a positive number',
    'number.max': 'Weight seems unrealistic (max 1000)',
    'any.required': 'Weight is required'
  }),
  date: Joi.date().iso().optional(),
  notes: Joi.string().max(500).optional()
});

export const updateBodyMeasurementsSchema = Joi.object({
  chest: Joi.number().positive().max(500).optional(),
  waist: Joi.number().positive().max(500).optional(),
  hips: Joi.number().positive().max(500).optional(),
  arms: Joi.number().positive().max(200).optional(),
  thighs: Joi.number().positive().max(200).optional(),
  bodyFat: Joi.number().min(0).max(100).optional(),
  date: Joi.date().iso().optional(),
  notes: Joi.string().max(500).optional()
});

export const uploadAvatarSchema = Joi.object({
  avatar: Joi.any().required().messages({
    'any.required': 'Avatar file is required'
  })
});

export const uploadProgressPhotosSchema = Joi.object({
  photos: Joi.array().items(Joi.any()).min(1).max(5).required().messages({
    'array.min': 'At least one photo is required',
    'array.max': 'Maximum 5 photos allowed',
    'any.required': 'Photos are required'
  }),
  date: Joi.date().iso().optional(),
  notes: Joi.string().max(500).optional()
});
