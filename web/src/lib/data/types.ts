export type Money = { amount: string; currencyCode: string };
export type Me = { userId: string; tenantId: string; capabilities: string[] };
export type Line = { sku: string; quantity: number; unitPrice: Money; lineAmount: Money };
export type Order = { id: string; code: string; status: string; createdBy: string; decidedBy: string; rejectReason: string; totalAmount: Money; lines: Line[]; allowedActions: string[]; createdAt: string; updatedAt: string };
export type Item = { sku: string; name: string; unitPrice: Money; available: number; onHand: number; reserved: number };
