import { Router, Request, Response } from 'express';
import { authenticateToken } from '@/middleware/auth';
import { validateWithJoi } from '@/middleware/validation';
import mealService from '@/services/mealService';
import Joi from 'joi';

const router = Router();

const createMealSchema = Joi.object({
  name: Joi.string().min(2).max(100).required(),
  mealType: Joi.string()
    .valid('BREAKFAST', 'LUNCH', 'DINNER', 'SNACK', 'MORNING_SNACK', 'AFTERNOON_SNACK', 'EVENING_SNACK', 'PRE_WORKOUT', 'POST_WORKOUT')
    .required(),
  mealDate: Joi.string().isoDate().required(),
  description: Joi.string().max(500).optional(),
  notes: Joi.string().max(500).optional(),
});

const updateMealSchema = Joi.object({
  name: Joi.string().min(2).max(100).optional(),
  mealType: Joi.string().valid('BREAKFAST', 'LUNCH', 'DINNER', 'SNACK', 'MORNING_SNACK', 'AFTERNOON_SNACK', 'EVENING_SNACK', 'PRE_WORKOUT', 'POST_WORKOUT').optional(),
  mealDate: Joi.string().isoDate().optional(),
  description: Joi.string().max(500).optional(),
  notes: Joi.string().max(500).optional(),
  isCompleted: Joi.boolean().optional(),
});

// Middleware for authentication
router.use(authenticateToken);

// GET /api/meals?date=YYYY-MM-DD
router.get('/', validateWithJoi(Joi.object({ date: Joi.string().isoDate() })), async (req: Request, res: Response) => {
  const userId = req.user!.id;
  const dateStr = req.query.date as string;
  const date = new Date(dateStr);
  const { meals, totals } = await mealService.listByDate(userId, date);
  res.json({ success: true, data: { meals, totals } });
});

// POST /api/meals
router.post('/', validateWithJoi(createMealSchema), async (req: Request, res: Response) => {
  const userId = req.user!.id;
  const { name, mealType, mealDate, description, notes } = req.body;
  const newMeal = await mealService.create({
    userId,
    name,
    mealType,
    mealDate: new Date(mealDate),
    description,
    notes,
  });
  res.status(201).json({ success: true, data: newMeal });
});

// PATCH /api/meals/:id
router.patch('/:id', validateWithJoi(updateMealSchema), async (req: Request, res: Response) => {
  const userId = req.user!.id;
  const { id } = req.params;
  const updates = req.body;
  if (updates.mealDate) {
    updates.mealDate = new Date(updates.mealDate);
  }
  const updatedMeal = await mealService.update(id, userId, updates);
  res.json({ success: true, data: updatedMeal });
});

// DELETE /api/meals/:id
router.delete('/:id', async (req: Request, res: Response) => {
  const userId = req.user!.id;
  const { id } = req.params;
  await mealService.remove(id, userId);
  res.status(204).send();
});

// POST /api/meals/:id/foods
router.post('/:id/foods', validateWithJoi(Joi.object({
  foodId: Joi.string().required(),
  quantity: Joi.number().positive().required(),
  servingSize: Joi.number().positive().required(),
  servingUnit: Joi.string().max(20).required(),
  calories: Joi.number().min(0).required(),
  protein: Joi.number().min(0).required(),
  carbs: Joi.number().min(0).required(),
  fat: Joi.number().min(0).required(),
  fiber: Joi.number().min(0).optional(),
  sugar: Joi.number().min(0).optional(),
  sodium: Joi.number().min(0).optional(),
  cholesterol: Joi.number().min(0).optional(),
  logDate: Joi.string().isoDate().required(),
  notes: Joi.string().max(300).optional()
})), async (req: Request, res: Response) => {
  const userId = req.user!.id;
  const { id: mealId } = req.params;
  const data = req.body;
  const log = await mealService.addFoodLog({ ...data, userId, mealId });
  res.status(201).json({ success: true, data: log });
});

// DELETE /api/meals/:mealId/foods/:foodLogId
router.delete('/:mealId/foods/:foodLogId', async (req: Request, res: Response) => {
  const userId = req.user!.id;
  const { mealId, foodLogId } = req.params;
  await mealService.removeFoodLog(foodLogId, mealId);
  res.status(204).send();
});

export default router;
