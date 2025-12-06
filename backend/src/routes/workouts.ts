import { Router, Request, Response } from 'express';
import Joi from 'joi';
import { authenticateToken } from '@/middleware/auth';
import { validateWithJoi } from '@/middleware/validation';
import prisma from '@/config/database';

const router = Router();

const createWorkoutSchema = Joi.object({
  name: Joi.string().min(2).max(100).required(),
  description: Joi.string().max(500).optional(),
  durationMinutes: Joi.number().min(5).max(300).required(),
  caloriesBurned: Joi.number().min(0).max(3000).optional(),
  date: Joi.string().isoDate().required(),
});

router.use(authenticateToken);

// List workouts by date range
router.get('/', async (req: Request, res: Response) => {
  const userId = req.user!.id;
  const { from, to } = req.query as any;
  const where: any = { userId };
  if (from && to) {
    where.date = { gte: new Date(from), lte: new Date(to) };
  }
  const workouts = await prisma.workoutLog.findMany({ where, orderBy: { date: 'desc' } });
  res.json({ success: true, data: workouts });
});

// Create workout log
router.post('/', validateWithJoi(createWorkoutSchema), async (req: Request, res: Response) => {
  const userId = req.user!.id;
  const input = req.body;
  const created = await prisma.workoutLog.create({
    data: {
      userId,
      name: input.name,
      description: input.description,
      durationMinutes: input.durationMinutes,
      caloriesBurned: input.caloriesBurned,
      date: new Date(input.date),
    },
  });
  res.status(201).json({ success: true, data: created });
});

// Delete workout log
router.delete('/:id', async (req: Request, res: Response) => {
  const userId = req.user!.id;
  const { id } = req.params;
  await prisma.workoutLog.delete({ where: { id } });
  res.status(204).send();
});

export default router;
