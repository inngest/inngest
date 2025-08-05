export type SavedQuery = {
  id: string;
  name: string;
  text: string;
  updatedOn: string; // ISO string
};

export type RecentQuery = Omit<SavedQuery, 'name'>;
