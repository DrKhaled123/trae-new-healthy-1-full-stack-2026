import Joi from 'joi';

export const searchFoodSchema = Joi.object({
  query: Joi.string().min(1).max(100).required().messages({
    'any.required': 'Search query is required',
    'string.min': 'Search query must be at least 1 character',
    'string.max': 'Search query cannot exceed 100 characters'
  }),
  limit: Joi.number().integer().min(1).max(50).default(10),
  offset: Joi.number().integer().min(0).default(0),
  category: Joi.string().optional(),
  brand: Joi.string().optional(),
  minCalories: Joi.number().min(0).optional(),
  maxCalories: Joi.number().min(0).optional(),
  sortBy: Joi.string().valid('name', 'calories', 'protein', 'carbs', 'fat', 'popularity').default('relevance'),
  sortOrder: Joi.string().valid('asc', 'desc').default('asc')
});

export const getFoodDetailsSchema = Joi.object({
  foodId: Joi.string().uuid().required().messages({
    'string.guid': 'Invalid food ID format',
    'any.required': 'Food ID is required'
  })
});

export const createCustomFoodSchema = Joi.object({
  name: Joi.string().min(1).max(100).required().messages({
    'any.required': 'Food name is required',
    'string.min': 'Food name must be at least 1 character',
    'string.max': 'Food name cannot exceed 100 characters'
  }),
  brand: Joi.string().max(100).optional(),
  category: Joi.string().max(50).required().messages({
    'any.required': 'Food category is required'
  }),
  description: Joi.string().max(500).optional(),
  servingSize: Joi.number().positive().required().messages({
    'any.required': 'Serving size is required',
    'number.positive': 'Serving size must be positive'
  }),
  servingUnit: Joi.string().max(20).required().messages({
    'any.required': 'Serving unit is required'
  }),
  nutrition: Joi.object({
    calories: Joi.number().min(0).required(),
    protein: Joi.number().min(0).required(),
    carbs: Joi.number().min(0).required(),
    fat: Joi.number().min(0).required(),
    fiber: Joi.number().min(0).optional(),
    sugar: Joi.number().min(0).optional(),
    sodium: Joi.number().min(0).optional(),
    cholesterol: Joi.number().min(0).optional(),
    saturatedFat: Joi.number().min(0).optional(),
    transFat: Joi.number().min(0).optional(),
    monounsaturatedFat: Joi.number().min(0).optional(),
    polyunsaturatedFat: Joi.number().min(0).optional(),
    vitaminA: Joi.number().min(0).optional(),
    vitaminC: Joi.number().min(0).optional(),
    vitaminD: Joi.number().min(0).optional(),
    vitaminE: Joi.number().min(0).optional(),
    vitaminK: Joi.number().min(0).optional(),
    thiamine: Joi.number().min(0).optional(),
    riboflavin: Joi.number().min(0).optional(),
    niacin: Joi.number().min(0).optional(),
    vitaminB6: Joi.number().min(0).optional(),
    folate: Joi.number().min(0).optional(),
    vitaminB12: Joi.number().min(0).optional(),
    calcium: Joi.number().min(0).optional(),
    iron: Joi.number().min(0).optional(),
    magnesium: Joi.number().min(0).optional(),
    phosphorus: Joi.number().min(0).optional(),
    potassium: Joi.number().min(0).optional(),
    zinc: Joi.number().min(0).optional(),
    copper: Joi.number().min(0).optional(),
    manganese: Joi.number().min(0).optional(),
    selenium: Joi.number().min(0).optional()
  }).required(),
  barcode: Joi.string().max(50).optional(),
  isPublic: Joi.boolean().default(false),
  tags: Joi.array().items(Joi.string().max(30)).optional()
});

export const updateCustomFoodSchema = Joi.object({
  name: Joi.string().min(1).max(100).optional(),
  brand: Joi.string().max(100).optional(),
  category: Joi.string().max(50).optional(),
  description: Joi.string().max(500).optional(),
  servingSize: Joi.number().positive().optional(),
  servingUnit: Joi.string().max(20).optional(),
  nutrition: Joi.object({
    calories: Joi.number().min(0).optional(),
    protein: Joi.number().min(0).optional(),
    carbs: Joi.number().min(0).optional(),
    fat: Joi.number().min(0).optional(),
    fiber: Joi.number().min(0).optional(),
    sugar: Joi.number().min(0).optional(),
    sodium: Joi.number().min(0).optional(),
    cholesterol: Joi.number().min(0).optional(),
    saturatedFat: Joi.number().min(0).optional(),
    transFat: Joi.number().min(0).optional(),
    monounsaturatedFat: Joi.number().min(0).optional(),
    polyunsaturatedFat: Joi.number().min(0).optional(),
    vitaminA: Joi.number().min(0).optional(),
    vitaminC: Joi.number().min(0).optional(),
    vitaminD: Joi.number().min(0).optional(),
    vitaminE: Joi.number().min(0).optional(),
    vitaminK: Joi.number().min(0).optional(),
    thiamine: Joi.number().min(0).optional(),
    riboflavin: Joi.number().min(0).optional(),
    niacin: Joi.number().min(0).optional(),
    vitaminB6: Joi.number().min(0).optional(),
    folate: Joi.number().min(0).optional(),
    vitaminB12: Joi.number().min(0).optional(),
    calcium: Joi.number().min(0).optional(),
    iron: Joi.number().min(0).optional(),
    magnesium: Joi.number().min(0).optional(),
    phosphorus: Joi.number().min(0).optional(),
    potassium: Joi.number().min(0).optional(),
    zinc: Joi.number().min(0).optional(),
    copper: Joi.number().min(0).optional(),
    manganese: Joi.number().min(0).optional(),
    selenium: Joi.number().min(0).optional()
  }).optional(),
  isPublic: Joi.boolean().optional(),
  tags: Joi.array().items(Joi.string().max(30)).optional()
});

export const deleteCustomFoodSchema = Joi.object({
  foodId: Joi.string().uuid().required().messages({
    'string.guid': 'Invalid food ID format',
    'any.required': 'Food ID is required'
  })
});

export const getFoodCategoriesSchema = Joi.object({
  includeCount: Joi.boolean().default(false)
});

export const scanBarcodeSchema = Joi.object({
  barcode: Joi.string().max(50).required().messages({
    'any.required': 'Barcode is required',
    'string.max': 'Barcode cannot exceed 50 characters'
  })
});

export const getPopularFoodsSchema = Joi.object({
  limit: Joi.number().integer().min(1).max(20).default(10),
  category: Joi.string().optional()
});

export const getRecentFoodsSchema = Joi.object({
  limit: Joi.number().integer().min(1).max(20).default(10)
});
