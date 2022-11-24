import Container from "./Container";

export default function OutTheBox() {
  return (
    <Container className="mt-20 mb-12">
      <h2 className="text-slate-50 font-medium text-2xl lg:text-4xl xl:text-5xl mb-4 tracking-tighter ">
        Everything you need - out of the box
      </h2>
      <p className="text-slate-300 font-light max-w-md lg:max-w-xl text-sm lg:text-base leading-5 lg:leading-7">
        We built all the features that you need to build powerful applications
        without having to re-invent the wheel.
      </p>
    </Container>
  );
}
