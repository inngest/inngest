export interface UserProfilePhotoUploaded {
  name: "user/profile.photo.uploaded";
  data: {
    /**
     * A signed public URL to access the profile photo the user wants to use.
     *
     * This is the un-optimized, un-checked version of the image.
     */
    url: string;
  };
  user: {
    email: string;
  };
  v?: string;
  ts?: number;
}

export type EventTriggers = UserProfilePhotoUploaded;

export type Args = {
  event: EventTriggers;
  steps: {
    [clientID: string]: any;
  };
};
