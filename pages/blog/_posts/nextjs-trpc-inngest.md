---
heading: "Building an Event Driven Video Processing Workflow with Next.js, tRPC, and Inngest"
subtitle: "How Badass Courses built a self-service video publishing workflow for Kent C. Dodds with AI generated transcripts and subtitles."
image: "/assets/blog/nextjs-trpc-inngest/epic-web-kent-c-dodds.png"
date: 2023-08-07
author: Joel Hooks
---

# End-to-End Video Processing and Transcription Workflow

If you’ve ever uploaded a video to a service like YouTube, you’ve experienced magic. It’s amazing. Astonishing what they’ve built. It feels seamless, and at every step of the way, you get feedback and have the ability to edit your video’s metadata while it processes in the background.

But what if you don’t want to use YouTube? There are many good reasons for this. YouTube serves ads to most users and offers recommendations for other content. You are forced into serving your media through their player, and offering users a custom experience is difficult.

In short, there are tradeoffs to all of the free miracle magic that YouTube provides!

So what needs to happen so that you can upload and serve video to your users?

1. Upload the original media.
2. Edit metadata, such as the video title and description.
3. Monitor progress and get feedback as the video is processed.
4. Convert the video media into appropriate formats for global streaming delivery.
5. Make your video as accessible as possible with transcripts and subtitles.

This might not look like a lot of steps, but it starts to get complicated and each step can potentially take 10s of seconds or even minutes to accomplish, depending on the size of the video files! You need to account for errors in the process, and you probably don’t want an error in the middle of the process to crash the entire workflow.

These are all of the problems that we faced when we were building a self-service video publishing for [Kent C. Dodds](https://twitter.com/kentcdodds) to use for his site [EpicWeb.dev](https://epicreact.dev).

Below is a real-world look into how we solved this for KCD, and created a repeatable durable process for him to use and upload video tips for his audience to learn from. There will be links to the actual production project on GitHub throughout the text below for you to explore.

We used a stack of excellent services to get the job done:

* **Vercel**: Hosting our Next.js application and serverless functions
* **Cloudinary**: Original media storage
* **Sanity**: Content Management System, but so much more. They see content as data and give us a ton of flexibility.
* **Mux**: excellent global video delivery and analytics
* **Deepgram**: AI Transcription using Whisper models for excellent high-quality results
* **OpenAI**: Just a little extra sauce.

All of this is glued together with **[Inngest, which provides consistent, reliable, resilient, and understandable workflows](https://inngest.com)** throughout the complex asynchronous process.

## The Problem

Kent is great to work with. He’s _extremely_ prolific, and we had a bottleneck in the process where he’d have to hand off media to the team and wait for us to process it manually for him. This is fine for longer format content like tutorials and courses, but Kent likes to post quick tips that he can share or use to answer questions directly at the moment on social media or in his Discord channel.

As a developer, Kent can withstand a measure of jankiness in the overall process 😅, but we wanted to make something that worked well and provided him with a nice experience, so he’d be stoked to post and share his tip videos.

These long-running processes are complicated and can often get messy, particularly if you want them to be user-friendly, but we had a good idea of how we wanted to build them.

## Uploading the Video Media

The first step is to get the media from Kent’s hard drive and onto the internet, in The Cloud, as a URL we can reference for the rest of the process.

![screenshot of the epic web dev tip upload form](/assets/blog/nextjs-trpc-inngest/epic-web-kent-c-dodds.png)

This took the shape of a [simple form that lives inside of our Next.js monorepo](https://github.com/skillrecordings/products/blob/2caf79b4adec32ac4dcd775b46c7544d6192cc0d/apps/epic-web/src/module-builder/create-tip-form.tsx) as a React component. The form requires two pieces of data from the creator:

* A title
* A video media file

When you hit the submit button on the form, it performs two actions. The first is to [upload the video file to Cloudinary as a multi-part upload](https://github.com/skillrecordings/products/blob/5320faede1b975a349eeb16e771b066eda310124/apps/epic-web/src/module-builder/cloudinary-video-uploader.ts) which will give us:

* A “forever” reference to the original media
* A DNS addressable URL to the original media that we can use for other aspects of the process, such as transcoding and generating transcripts.

An alternative to Cloudinary might be to upload to Amazon’s S3 or some other similar blob storage service like [Upload Thing](https://uploadthing.com/).

Wherever it ends up, the result is a URL that we can reference, so once that’s done, we can move to the next step in the process.

## Use the Video Media URL to Create the Tip Resource

With the video media uploaded, we send the title and the media URL to the backend with a tRPC mutation.

[tRPC](https://trpc.io/) is a library that gives seamless type safety with your Next.js serverless function API. It uses [react-query](https://tanstack.com/query/latest/docs/react) under the hood and gives you a nice full-stack developer experience with TypeScript.

Calling the [mutation looks like this](https://github.com/skillrecordings/products/blob/2caf79b4adec32ac4dcd775b46c7544d6192cc0d/apps/epic-web/src/module-builder/create-tip-form.tsx#L66C9-L78C10):

```typescript
 const {mutate: createTip} = trpc.tips.create.useMutation()

  const handleSubmit = async (values?: any, event?: BaseSyntheticEvent) => {
    try {
      if (fileType && fileContents) {
        setState('uploading')
        const uploadResponse: {secure_url: string} = await processFile(
          fileContents,
          (progress) => {
            setProgress(progress)
          },
        )

        setState('success')

        console.log({values})

        createTip(
          {
            s3Url: uploadResponse.secure_url,
            fileName,
            title: values.title,
          },
          {
            onSettled: (data) => {
              console.log('tip creation settled', data)
              router.push(`/creator/tips/${data?.slug}`)
            },
          },
        )
      }
    } catch (err) {
      setState('error')
      console.log('error is', err)
    }
  }
```

This mutation effectively submits our form and sends the data to the server so we can safely and securely kick off our video processing workflow.

### The tRPC Mutation Creates the Sanity Resource

tRPC uses routers to group together related functionality and provide you with a convenient API for executing serverless functions. The router contains queries and mutations which accept input. The input is validated and strongly typed using [Zod](https://zod.dev/), a schema validation and type generation library.

Here’s [the full tRPC mutation](https://github.com/skillrecordings/products/blob/65dde5644242ec089bcefedc966e912ca6abf8f2/apps/epic-web/src/trpc/routers/tips.ts#L17C1-L93C8):

```typescript
export const tipsRouter = router({
  create: publicProcedure
    .input(
      z.object({
        s3Url: z.string(),
        fileName: z.string().nullable(),
        title: z.string(),
      }),
    )
    .mutation(async ({ctx, input}) => {
      // create a video resource, which should trigger the process of uploading to
      // mux and ordering a transcript because of the active webhook
      const token = await getToken({req: ctx.req})
      const ability = getCurrentAbility({
        user: UserSchema.parse(token),
      })

      // use CASL rbac to check if the user can create content
      if (ability.can('create', 'Content')) {
        // create the video resource object in Sanity
        const newVideoResource = await sanityWriteClient.create({
          _id: `videoResource-${v4()}`,
          _type: 'videoResource',
          state: 'new',
          title: input.fileName,
          originalMediaUrl: input.s3Url,
        })

        if (newVideoResource._id) {
          // control the id that is used so we can reference it immediately
          const id = v4()

          const nanoid = customAlphabet(
            '1234567890abcdefghijklmnopqrstuvwxyz',
            5,
          )

          // create the Tip resource in sanity with the video resource attached
          const tipResource = await sanityWriteClient.create({
            _id: `tip-${id}`,
            _type: 'tip',
            state: 'new',
            title: input.title,
            slug: {
              // since title is unique, we can use it as the slug with a random string
              current: `${slugify(input.title)}~${nanoid()}`,
            },
            resources: [
              {
                _key: v4(),
                _type: 'reference',
                _ref: newVideoResource._id,
              },
            ],
          })

          // load the complete tip from sanity so we can return it
          // we are reloading it because the query for `getTip` "normalizes"
          // the data and that's what we expect client-side
          const tip = await getTip(tipResource.slug.current)

          await inngest.send({
            name: 'tip/video.uploaded',
            data: {
              tipId: tip._id,
              videoResourceId: newVideoResource._id,
            },
          })

          return tip
        } else {
          throw new Error('Could not create video resource')
        }
      } else {
        throw new Error('Unauthorized')
      }
    }),
```

The mutation creates multiple resources in Sanity, which is a headless content management system (CMS) that treats your content as data and stores it for later querying and retrieval. For Kent’s tips, we are creating two documents:

* **Video Resource** represents an “immutable” reference to the original video media. Every video we upload is unique and points to a URL of the original video media.
* **Tip Resource** that has a reference to the Video Resource. The Tip Resource contains metadata around the tip itself, such as the title, description, and other similar information.

Storing these two resources in our content management system means they are safe and secure and ready for the rest of the process to proceed.

The next steps are:

* Convert the video to an adaptive streaming format and use a global CDN for distribution. We love Mux.
* Create transcripts for the video.
* Create subtitles for the video.
* Generate some placeholder text for the video using OpenAI large language models

There are a lot of moving parts, and keeping it consistent, organized, and reliably error resistant is a huge challenge.

Here’s where Inngest starts to shine ✨

## Process the new Tip Video

When the Tip has been created and everything is ready to go, we [send an event to Inngest to start our workflow from within the tRPC mutation](https://github.com/skillrecordings/products/blob/65dde5644242ec089bcefedc966e912ca6abf8f2/apps/epic-web/src/trpc/routers/tips.ts#L78-L84).

```typescript
await inngest.send({
  name: 'tip/video.uploaded',
  data: {
    tipId: tip._id,
    videoResourceId: newVideoResource._id,
  },
})
```

We also return the newly created Tip to the client immediately so that the user can have a visual display of the current state and feel comfortable that the helper robots are behind the scenes making the magic happen.

Inngest picks up the event and executes our own API routes to run the workflow. Check out the [full code for the workflow here on GitHub](https://github.com/skillrecordings/products/blob/d7cfbfab3e4339fb3d1bbfcf81cf97b79819b9ad/apps/epic-web/src/pages/api/inngest.ts#L132-L267). It’s complex! It’s complex because the process is complex, but when you look at this code each step is named and well-defined giving us a readable process that we can understand.

This makes the initial development, maintenance, and debugging issues so much simpler than other approaches.

Let's go through the workflow to understand what’s happening.

### Update the Status of our Tips

Along the way, we want to keep the status of the tip updated. This allows us to visually present to the creator the current status of the tip in the process. Letting them know along the way is hugely important. Otherwise, they will think your process is broken and be sad about it.

Inside of our [multi-step Inngest function, we will first update the status](https://github.com/skillrecordings/products/blob/65dde5644242ec089bcefedc966e912ca6abf8f2/apps/epic-web/src/pages/api/inngest.ts#L136-L143):

```typescript
await step.run('Update Tip Status', async () => {
  return await sanityWriteClient
    .patch(event.data.tipId)
    .set({
      state: 'processing',
    })
    .commit()
})
```

We don’t like sad creators! Because our Inngest workflow is well organized we can discreetly update our data which is then used by our front-end to display progress and status to the user.

Looking at the workflow, you’ll also notice that each step is contained, and we try to keep our Inngest steps to a **single piece of work** instead of cramming in several actions into each step. This helps make our workflow more durable and lets Inngest re-run steps more safely. It also helps when something goes wrong and you need to debug.

### Create a new Asset in Mux

This [step sends our original media URL to Mux](https://github.com/skillrecordings/products/blob/d7cfbfab3e4339fb3d1bbfcf81cf97b79819b9ad/apps/epic-web/src/pages/api/inngest.ts#L145-L153) to be converted into the appropriate format for global distribution. Mux is an awesome video service like “Stripe for Video”.

```typescript
const newMuxAsset = await step.run("Create a Mux Asset", async () => {
  const videoResource = await getVideoResource(event.data.videoResourceId)
  const {originalMediaUrl, muxAsset, duration} = videoResource
  return await createMuxAsset({
    originalMediaUrl,
    muxAsset,
    duration,
  })
})
```

They provide excellent delivery of the media, analytics, and have a wonderful video player.

Mux is the first stop for our freshly uploaded video and all that is required is the URL to the original media.

### Update the Tip in Sanity

When we send the video to Mux, we create a Mux Asset, and we can [associate that asset with the Video Resource we created in Sanity](https://github.com/skillrecordings/products/blob/d7cfbfab3e4339fb3d1bbfcf81cf97b79819b9ad/apps/epic-web/src/pages/api/inngest.ts#L155-L166). This is attached to the Video Resource and not the tip because it represents the video media.

```typescript
await step.run("Sync Asset with Sanity", async () => {
  const videoResource = await getVideoResource(event.data.videoResourceId)
  const {duration: assetDuration, ...muxAsset} = newMuxAsset

  return await sanityWriteClient
    .patch(videoResource._id)
    .set({
      duration: assetDuration,
      muxAsset,
    })
    .commit()
})
```

With all of that in place we can display the tip in the front-end for the creator once Mux has finished processing the video.

Since we care about learners, we also want to ensure the videos are accessible by creating transcriptions and subtitles.

### Order a Transcript via Deepgram

Deepgram uses Whisper to generate high-quality AI-fueled transcripts. For the purposes of our use case, troubleshooting, and development of the serverside aspects of this transcript generation process, we chose Cloudflare Workers. In this step, we [trigger that process to run outside of our Next.js app](https://github.com/skillrecordings/products/blob/65dde5644242ec089bcefedc966e912ca6abf8f2/apps/epic-web/src/pages/api/inngest.ts#L168-L180):

```typescript
await step.run("Initiate Transcript Order via Deepgram", async () => {
  const videoResource = await getVideoResource(event.data.videoResourceId)
  const {originalMediaUrl, _id} = videoResource
  return await fetch(
    `https://deepgram-wrangler.skillstack.workers.dev/transcript?videoUrl=${originalMediaUrl}&videoResourceId=${_id}`,
    {
      method: "POST",
      headers: {
        "Content-Type": "application/json",
      },
    }
  )
})
```

Now we wait…

### Wait for the Transcript to be complete

Back in our Next.js app, our Inngest [workflow is waiting for the next step in the process to complete](https://github.com/skillrecordings/products/blob/d7cfbfab3e4339fb3d1bbfcf81cf97b79819b9ad/apps/epic-web/src/pages/api/inngest.ts#L184-L187) and once that happens we send an event from inside our Cloudflare Worker:

```typescript
const transcript = await step.waitForEvent("tip/video.transcript.created", {
  match: "data.videoResourceId",
  timeout: "1h",
})
```

Depending on the length of the video, this process can take a while, but our workflow will patiently wait for it to finish before proceeding. You can set custom timeouts when waiting for events inside Inngest workflows from seconds to hours, which is very handy in cases where a timeout might occur and you want the workflow to continue anyway.

Sending the event is simple from Cloudflare (or anywhere!) with Inngest. In this case, we [use the Inngest Cloudflare library and send the event](https://github.com/skillrecordings/video-text-processing-worker/blob/7be64ee1902d4bfa0a438208074eacc44cbc5ba7/src/transcriptComplete.ts#L80-L83) to let the workflow know it can proceed.

```typescript
const inngestResponse = await inngest.send({
 name: 'tip/video.transcript.created',
 data,
})
```

EZ

### Update the Tip Video Resource in Sanity with the Transcript

Our [workflow picks up the notification, takes the payload and updates the Video Resource in Sanity](https://github.com/skillrecordings/products/blob/65dde5644242ec089bcefedc966e912ca6abf8f2/apps/epic-web/src/pages/api/inngest.ts#L190-L200) so that our video will have a full-transcript. We will also attach the srt to the video resource so that it will be completed and reviewed, and edited later as needed.

```typescript
await step.run("Update Video Resource with Transcript", async () => {
  return await sanityWriteClient
    .patch(event.data.videoResourceId)
    .set({
      transcript: {
        text: transcript.data.transcript.text,
        srt: transcript.data.transcript.srt,
      },
    })
    .commit()
})
```

At this point we need to take a little detour and kickoff a sub-workflow to make sure everything is timed just right.

### Kick-off another workflow to attach the subtitles to the video in Mux

Mux provides an API to update our Mux Asset with the srt, which is very nice with the Mux Player because the subtitles will show up automatically without any additional configuration.

```typescript
await step.run("Notify SRT is Ready to Add to Mux Asset", async () => {
  return await inngest.send({
    name: "tip/video.srt.ready",
    data: {
      muxAssetId: newMuxAsset.muxAssetId,
      videoResourceId: event.data.videoResourceId,
      srt: transcript.data.transcript.srt,
    },
  })
})
```

This part is tricky since the Deepgram transcription can sometimes be faster than the Mux processing. If Mux hasn’t finished processing the video and you send a transcript, it throws an error.

[Inngest makes it easy to kick off a resilient sub-workflow](https://github.com/skillrecordings/products/blob/65dde5644242ec089bcefedc966e912ca6abf8f2/apps/epic-web/src/pages/api/inngest.ts#L60-L130) so that we don’t need to block the rest of our main workflow to finalize the Mux Asset

```typescript
const addSrtToMuxAsset = inngest.createFunction(
  {name: "Add SRT to Mux Asset"},
  {event: "tip/video.srt.ready"},
  async ({event, step}) => {
    const muxAssetStatus = await step.run(
      "Check if Mux Asset is Ready",
      async () => {
        const {Video} = new Mux()
        const {status} = await Video.Assets.get(event.data.muxAssetId)
        return status
      }
    )

    await step.run("Update Video Resource Status", async () => {
      return await sanityWriteClient
        .patch(event.data.videoResourceId)
        .set({
          state: muxAssetStatus,
        })
        .commit()
    })

    if (muxAssetStatus === "ready") {
      await step.run(
        "Check for existing subtitles in Mux and remove if found",
        async () => {
          const {Video} = new Mux()
          const {tracks} = await Video.Assets.get(event.data.muxAssetId)

          const existingSubtitle = tracks?.find(
            (track: any) => track.name === "English"
          )

          if (existingSubtitle) {
            return await Video.Assets.deleteTrack(
              event.data.muxAssetId,
              existingSubtitle.id
            )
          } else {
            return "No existing subtitle found."
          }
        }
      )

      await step.run("Update Mux with SRT", async () => {
        const {Video} = new Mux()
        return await Video.Assets.createTrack(event.data.muxAssetId, {
          url: `https://www.epicweb.dev/api/videoResource/${event.data.videoResourceId}/srt`,
          type: "text",
          text_type: "subtitles",
          closed_captions: false,
          language_code: "en-US",
          name: "English",
          passthrough: "English",
        })
      })

      // await step.run('Notify in Slack', async () => {
      //
      // })
    } else {
      await step.sleep(60000)
      await step.run("Re-run After Cooldown", async () => {
        return await inngest.send({
          name: "tip/video.srt.ready",
          data: event.data,
        })
      })
    }
  }
)
```

This sub-workflow checks the status of the video in Mux, sleeps for a bit if it is still processing, and then tries again until it gets the green light.

Works like a charm and doesn’t block the rest of our workflow.

### Send the transcript to OpenAI to get some suggestions

This is an experiment that we are excited to keep exploring, but now that we’ve got a complete transcript, we send it into OpenAI to get a placeholder for body text, short descriptions, alternative titles, tweets, emails, keywords, and additional article ideas.

```typescript
await step.run("Send Transcript for LLM Suggestions", async () => {
  // this step initiates a call to worker and then doesn't bother waiting for a response
  // the sleep is just a small hedge to make sure we don't close the connection immediately
  // but the worker seems to run just fine if we don't bother waiting for a response
  // this isn't great, really, but waiting for the worker response times it out consistently
  // even with shorter content
  fetch(
    `https://deepgram-wrangler.skillstack.workers.dev/tipMetadataLLM?videoResourceId=${event.data.videoResourceId}`,
    {
      method: "POST",
      headers: {
        "Content-Type": "application/json",
      },
      body: JSON.stringify({
        transcript: transcript.data.transcript.text,
        tipId: event.data.tipId,
      }),
    }
  )
  await sleep(1000)
  return "Transcript sent to LLM"
})
```

This isn’t perfect most of the time, but it is an exciting way to get the creative juices flowing and take the next steps.

We are excited to apply more [ideas that Maggie Appleton outlines in her excellent Language Model Sketches](https://maggieappleton.com/lm-sketchbook). The Daemons, in particular, could be interesting in this context.

We used langchain’s tooling to craft our prompt for the gpt-3.5-turbo-16k model and sent it a prompt that requests that the results be delivered in JSON in a specific format.

It was surprising that it was able to do this, but it is consistent with it. We added a fallback to use gpt-4 if the json doesn’t parse as an attempt to “heal” the result.

Very cool.

Once that’s complete we use the Inngest Cloudflare client to send an event and pick the process back up on the Next.js side of the fence.

### Apply the LLM suggestions to the Tip Resource

This is the end of the workflow and we can now update the status of the Tip and apply the LLM suggestion to the Tip Resource in Sanity.

```typescript
const llmResponse = await step.waitForEvent(
  "tip/video.llm.suggestions.created",
  {
    match: "data.videoResourceId",
    timeout: "1h",
  }
)

if (llmResponse) {
  await step.run("Update Tip with Generated Text", async () => {
    const title = llmResponse.data.llmSuggestions?.titles?.[0]
    const body = llmResponse.data.llmSuggestions?.body
    const description = llmResponse.data.llmSuggestions?.descriptions?.[0]
    return await sanityWriteClient
      .patch(event.data.tipId)
      .set({
        title,
        description,
        body,
        state: "reviewing",
      })
      .commit()
  })
  return {llmSuggestions: llmResponse.data.llmSuggestions, transcript}
} else {
  return {transcript, llmSuggestions: null}
}
```

Done.

## Wrapping up and Next Steps

Overall, this complex process was made coherent because Inngest lets us define the workflow in a way that makes sense and “just works.”

At every step of the way, we can log into the Inngest cloud dashboard and see the status of our workflow, debug the steps, and have a clear understanding of what’s going on with our process.

We used Cloudflare for the longer timeouts for these processes, but you might also have success using edge functions and the App Router in Next.js.

This is just the start. From here, with the base process in place, we can start to consider so many exciting options and improvements down the road and take this base process to build up more complex content publishing pipelines.
