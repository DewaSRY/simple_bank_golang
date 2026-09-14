export type AccountList = {
  id: number;
  name: string;
  number: string;
};

export type DepositeStep = "account" | "details" | "preview";

export type DepositeDetails = {
  amount: number;
  description: string;
};
