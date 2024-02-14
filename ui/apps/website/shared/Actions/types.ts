export type ActionCategory = {
  id: string;
  name: string;
  actions: Action[];
};

export type Action = {
  dsn: string;
  title: string;
  description: string;
};

export type ActionWithCategoryName = Action & {
  category: string;
};
