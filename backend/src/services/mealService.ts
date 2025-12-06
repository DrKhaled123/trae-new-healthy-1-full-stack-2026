import prisma from '@/config/database';

export type MealType = 'BREAKFAST' | 'LUNCH' | 'DINNER' | 'SNACK' | 'MORNING_SNACK' | 'AFTERNOON_SNACK' | 'EVENING_SNACK' | 'PRE_WORKOUT' | 'POST_WORKOUT';

export interface CreateMealInput {
  userId: string;
  name: string;
  mealType: MealType;
  mealDate: Date;
  description?: string;
  notes?: string;
}

export interface UpdateMealInput {
  name?: string;
  mealType?: MealType;
  mealDate?: Date;
  description?: string;
  notes?: string;
  isCompleted?: boolean;
}

export const mealService = {
  async listByDate(userId: string, date: Date) {
    const start = new Date(date);
    start.setHours(0, 0, 0, 0);
    const end = new Date(date);
    end.setHours(23, 59, 59, 999);

    const meals = await prisma.meal.findMany({
      where: { userId, mealDate: { gte: start, lte: end } },
      include: { foodLogs: true },
      orderBy: [{ mealDate: 'asc' }],
    });

    const totals = meals.reduce(
      (acc, m) => {
        acc.calories += m.calories;
        acc.protein += m.protein;
        acc.carbs += m.carbs;
        acc.fat += m.fat;
        return acc;
      },
      { calories: 0, protein: 0, carbs: 0, fat: 0 }
    );

    return { meals, totals };
  },

  async create(input: CreateMealInput) {
    return prisma.meal.create({ data: input });
  },

  async update(mealId: string, userId: string, input: UpdateMealInput) {
    return prisma.meal.update({
      where: { id: mealId },
      data: input,
    });
  },

  async remove(mealId: string, userId: string) {
    await prisma.foodLog.deleteMany({ where: { mealId } });
    return prisma.meal.delete({ where: { id: mealId } });
  },

  async addFoodLog(params: {
    userId: string;
    mealId: string;
    foodId?: string; // optional for custom entry
    isCustom?: boolean;
    name?: string; // for custom
    quantity: number;
    servingSize: number;
    servingUnit?: string;
    calories: number;
    protein: number;
    carbs: number;
    fat: number;
    fiber?: number;
    sugar?: number;
    sodium?: number;
    cholesterol?: number;
    logDate: Date;
    notes?: string;
  }) {
    const log = await prisma.foodLog.create({ data: params });

    // Recalculate meal totals
    const agg = await prisma.foodLog.aggregate({
      _sum: { calories: true, protein: true, carbs: true, fat: true },
      where: { mealId: params.mealId },
    });

    await prisma.meal.update({
      where: { id: params.mealId },
      data: {
        calories: agg._sum.calories ?? 0,
        protein: agg._sum.protein ?? 0,
        carbs: agg._sum.carbs ?? 0,
        fat: agg._sum.fat ?? 0,
      },
    });

    return log;
  },

  async removeFoodLog(foodLogId: string, mealId: string) {
    await prisma.foodLog.delete({ where: { id: foodLogId } });

    const agg = await prisma.foodLog.aggregate({
      _sum: { calories: true, protein: true, carbs: true, fat: true },
      where: { mealId },
    });

    await prisma.meal.update({
      where: { id: mealId },
      data: {
        calories: agg._sum.calories ?? 0,
        protein: agg._sum.protein ?? 0,
        carbs: agg._sum.carbs ?? 0,
        fat: agg._sum.fat ?? 0,
      },
    });
  },
};

export default mealService;
