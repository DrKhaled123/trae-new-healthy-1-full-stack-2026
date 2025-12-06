import { Router, Request, Response } from 'express';
import Joi from 'joi';
import { authenticateToken } from '@/middleware/auth';
import { validateWithJoi } from '@/middleware/validation';
import mealService, { MealType } from '@/services/mealService';

const router = Router();

// Schemas
const dateQuerySchema = Joi.object({
  date: Joi.string().isoDate().required(),
});

const createMealSchema = Joi.object({
  name: Joi.string().min(2).max(100).required(),
  mealType: Joi.string()
    .valid(
      'BREAKFAST','LUNCH','DINNER','SNACK','MORNING_SNACK','AFTERNOON_SNACK','EVENING_SNACK','PRE_WORKOUT','POST_WORKOUT'
    )
    .required(),
  mealDate: Joi.string().isoDate().required(),
  description: Joi.string().max(500).optional(),
  notes: Joi.string().max(500).optional(),
});

const updateMealSchema = Joi.object({
  name: Joi.string().min(2).max(100).optional(),
  mealType: Joi.string().valid(
    'BREAKFAST','LUNCH','DINNER','SNACK','MORNING_SNACK','AFTERNOON_SNACK','EVENING_SNACK','PRE_WORKOUT','POST_WORKOUT'
  ).optional(),
  mealDate: Joi.string().isoDate().optional(),
  description: Joi.string().max(500).optional(),
  notes: Joi.string().max(500).optional(),
  isCompleted: Joi.boolean().optional(),
});

const addFoodLogSchema = Joi.object({
  foodId: Joi.string().optional(),
  isCustom: Joi.boolean().default(false),
  name: Joi.string().when('isCustom', { is: true, then: Joi.required(), otherwise: Joi.optional() }),
  quantity: Joi.number().positive().required(),
  servingSize: Joi.number().positive().required(),
  servingUnit: Joi.string().max(20).default('g'),
  calories: Joi.number().min(0).required(),
  protein: Joi.number().min(0).required(),
  carbs: Joi.number().min(0).required(),
  fat: Joi.number().min(0).required(),
  fiber: Joi.number().min(0).optional(),
  sugar: Joi.number().min(0).optional(),
  sodium: Joi.number().min(0).optional(),
  cholesterol: Joi.number().min(0).optional(),
  logDate: Joi.string().isoDate().required(),
  notes: Joi.string().max(300).optional(),
});

// Auth for all routes
router.use(authenticateToken);

// GET /api/meals?date=YYYY-MM-DD
router.get('/', validateWithJoi(dateQuerySchema, 'query'), async (req: Request, res: Response) => {
  const userId = req.user!.id;
  const { date } = req.query as { date: string };
  const result = await mealService.listByDate(userId, new Date(date));
  res.json({ success: true, data: result });
});

// POST /api/meals
router.post('/', validateWithJoi(createMealSchema), async (req: Request, res: Response) => {
  const userId = req.user!.id;
  const { name, mealType, mealDate, description, notes } = req.body as {
    name: string; mealType: MealType; mealDate: string; description?: string; notes?: string;
  };

  const created = await mealService.create({
    userId,
    name,
    mealType,
    mealDate: new Date(mealDate),
    description,
    notes,
  });
  res.status(201).json({ success: true, data: created });
});

// PATCH /api/meals/:id
router.patch('/:id', validateWithJoi(updateMealSchema), async (req: Request, res: Response) => {
  const userId = req.user!.id;
  const { id } = req.params;
  const input = req.body;

  // Convert date if present
  if (input.mealDate) input.mealDate = new Date(input.mealDate);

  const updated = await mealService.update(id, userId, input);
  res.json({ success: true, data: updated });
});

// DELETE /api/meals/:id
router.delete('/:id', async (req: Request, res: Response) => {
  const userId = req.user!.id;
  const { id } = req.params;
  await mealService.remove(id, userId);
  res.status(204).send();
});

// POST /api/meals/:id/foods
router.post('/:id/foods', validateWithJoi(addFoodLogSchema), async (req: Request, res: Response) => {
  const userId = req.user!.id;
  const { id } = req.params;
  const input = req.body;

  const created = await mealService.addFoodLog({
    userId,
    mealId: id,
    foodId: input.foodId,
    isCustom: input.isCustom,
    name: input.name,
    quantity: input.quantity,
    servingSize: input.servingSize,
    servingUnit: input.servingUnit,
    calories: input.calories,
    protein: input.protein,
    carbs: input.carbs,
    fat: input.fat,
    fiber: input.fiber,
    sugar: input.sugar,
    sodium: input.sodium,
    cholesterol: input.cholesterol,
    logDate: new Date(input.logDate),
    notes: input.notes,
  });

  res.status(201).json({ success: true, data: created });
});

// DELETE /api/meals/:mealId/foods/:foodLogId
router.delete('/:mealId/foods/:foodLogId', async (req: Request, res: Response) => {
  const { mealId, foodLogId } = req.params;
  await mealService.removeFoodLog(foodLogId, mealId);
  res.status(204).send();
});

export default router;
