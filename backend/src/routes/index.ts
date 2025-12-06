import { Router } from 'express';
import health from './health';
import meals from './meals';
import workouts from './workouts';
import progress from './progress';
import auth from './auth';

const router = Router();

router.use('/health', health);
router.use('/meals', meals);
router.use('/workouts', workouts);
router.use('/progress', progress);
router.use('/auth', auth);

export default router;
