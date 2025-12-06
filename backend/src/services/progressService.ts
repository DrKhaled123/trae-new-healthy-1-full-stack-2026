import prisma from '@/config/database';

export interface CreateWeightInput {
  userId: string;
  weight: number; // kg
  bodyFat?: number; // %
  muscle?: number; // kg
  notes?: string;
  recordDate: Date;
}

export interface CreateMeasurementInput {
  userId: string;
  chest?: number;
  waist?: number;
  hips?: number;
  arms?: number;
  thighs?: number;
  calves?: number;
  neck?: number;
  shoulders?: number;
  forearms?: number;
  wrist?: number;
  bodyFatCaliper?: number;
  notes?: string;
  measurementDate: Date;
}

export const progressService = {
  async listWeights(userId: string, from?: Date, to?: Date) {
    return prisma.progressRecord.findMany({
      where: {
        userId,
        recordDate: from && to ? { gte: from, lte: to } : undefined,
      },
      orderBy: { recordDate: 'asc' },
      select: { id: true, recordDate: true, weight: true, bodyFat: true, muscle: true, notes: true },
    });
  },

  async addWeight(input: CreateWeightInput) {
    return prisma.progressRecord.create({
      data: input,
      select: { id: true, recordDate: true, weight: true, bodyFat: true, muscle: true, notes: true },
    });
  },

  async deleteWeight(userId: string, id: string) {
    return prisma.progressRecord.delete({ where: { id } });
  },

  async listMeasurements(userId: string, from?: Date, to?: Date) {
    return prisma.bodyMeasurement.findMany({
      where: {
        userId,
        measurementDate: from && to ? { gte: from, lte: to } : undefined,
      },
      orderBy: { measurementDate: 'asc' },
    });
  },

  async addMeasurement(input: CreateMeasurementInput) {
    return prisma.bodyMeasurement.create({ data: input });
  },

  async deleteMeasurement(userId: string, id: string) {
    return prisma.bodyMeasurement.delete({ where: { id } });
  },
};

export default progressService;
