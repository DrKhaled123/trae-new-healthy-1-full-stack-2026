import { Router, Request, Response } from 'express';
import Joi from 'joi';
import { authenticateToken } from '@/middleware/auth';
import { validateWithJoi } from '@/middleware/validation';
import progressService from '@/services/progressService';

const router = Router();

const rangeSchema = Joi.object({
  from: Joi.string().isoDate().optional(),
  to: Joi.string().isoDate().optional(),
});

const addWeightSchema = Joi.object({
  weight: Joi.number().min(20).max(400).required(),
  bodyFat: Joi.number().min(0).max(70).optional(),
  muscle: Joi.number().min(0).max(200).optional(),
  notes: Joi.string().max(300).optional(),
  recordDate: Joi.string().isoDate().required(),
});

const addMeasurementSchema = Joi.object({
  chest: Joi.number().min(0).max(200).optional(),
  waist: Joi.number().min(0).max(200).optional(),
  hips: Joi.number().min(0).max(200).optional(),
  arms: Joi.number().min(0).max(100).optional(),
  thighs: Joi.number().min(0).max(150).optional(),
  calves: Joi.number().min(0).max(100).optional(),
  neck: Joi.number().min(0).max(80).optional(),
  shoulders: Joi.number().min(0).max(200).optional(),
  forearms: Joi.number().min(0).max(80).optional(),
  wrist: Joi.number().min(0).max(40).optional(),
  bodyFatCaliper: Joi.number().min(0).max(70).optional(),
  notes: Joi.string().max(300).optional(),
  measurementDate: Joi.string().isoDate().required(),
});

router.use(authenticateToken);

router.get('/weights', validateWithJoi(rangeSchema, 'query'), async (req: Request, res: Response) => {
  const userId = req.user!.id;
  const { from, to } = req.query as any;
  const data = await progressService.listWeights(userId, from ? new Date(from) : undefined, to ? new Date(to) : undefined);
  res.json({ success: true, data });
});

router.post('/weights', validateWithJoi(addWeightSchema), async (req: Request, res: Response) => {
  const userId = req.user!.id;
  const input = req.body;
  const created = await progressService.addWeight({
    userId,
    weight: input.weight,
    bodyFat: input.bodyFat,
    muscle: input.muscle,
    notes: input.notes,
    recordDate: new Date(input.recordDate),
  });
  res.status(201).json({ success: true, data: created });
});

router.delete('/weights/:id', async (req: Request, res: Response) => {
  const userId = req.user!.id;
  const { id } = req.params;
  await progressService.deleteWeight(userId, id);
  res.status(204).send();
});

router.get('/measurements', validateWithJoi(rangeSchema, 'query'), async (req: Request, res: Response) => {
  const userId = req.user!.id;
  const { from, to } = req.query as any;
  const data = await progressService.listMeasurements(userId, from ? new Date(from) : undefined, to ? new Date(to) : undefined);
  res.json({ success: true, data });
});

router.post('/measurements', validateWithJoi(addMeasurementSchema), async (req: Request, res: Response) => {
  const userId = req.user!.id;
  const input = req.body;
  const created = await progressService.addMeasurement({
    userId,
    chest: input.chest,
    waist: input.waist,
    hips: input.hips,
    arms: input.arms,
    thighs: input.thighs,
    calves: input.calves,
    neck: input.neck,
    shoulders: input.shoulders,
    forearms: input.forearms,
    wrist: input.wrist,
    bodyFatCaliper: input.bodyFatCaliper,
    notes: input.notes,
    measurementDate: new Date(input.measurementDate),
  });
  res.status(201).json({ success: true, data: created });
});

router.delete('/measurements/:id', async (req: Request, res: Response) => {
  const userId = req.user!.id;
  const { id } = req.params;
  await progressService.deleteMeasurement(userId, id);
  res.status(204).send();
});

export default router;
