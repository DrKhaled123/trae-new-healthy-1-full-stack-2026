import prisma from '@/config/database';
import { ActivityLevel, GoalType } from '@prisma/client';

export interface CreateGoalInput {
  userId: string;
  name: string;
  description?: string;
  goalType: GoalType;
  targetCalories?: number;
  targetProtein?: number;
  targetCarbs?: number;
  targetFat?: number;
  targetFiber?: number;
  targetWater?: number;
  activityLevel: ActivityLevel;
  startDate: Date;
  targetDate?: Date;
}

export interface UpdateGoalInput extends Partial<CreateGoalInput> {
  isActive?: boolean;
  isCompleted?: boolean;
}

export const goalService = {
  async list(userId: string) {
    return prisma.nutritionGoal.findMany({ where: { userId }, orderBy: { createdAt: 'desc' } });
  },

  async active(userId: string) {
    return prisma.nutritionGoal.findFirst({ where: { userId, isActive: true }, orderBy: { createdAt: 'desc' } });
  },

  async create(input: CreateGoalInput) {
    // Deactivate existing active goals
    await prisma.nutritionGoal.updateMany({ where: { userId: input.userId, isActive: true }, data: { isActive: false } });

    return prisma.nutritionGoal.create({ data: { ...input, isActive: true } });
  },

  async update(id: string, userId: string, input: UpdateGoalInput) {
    return prisma.nutritionGoal.update({ where: { id }, data: input });
  },

  async remove(id: string, userId: string) {
    return prisma.nutritionGoal.delete({ where: { id } });
  },
};

export default goalService;
