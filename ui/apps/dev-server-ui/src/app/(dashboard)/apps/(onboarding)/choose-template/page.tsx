import FrameworkList from '@/components/Onboarding/FrameworkList';
import templatesData from '@/components/Onboarding/templates.json';

export default function Page() {
  return (
    <FrameworkList
      frameworksData={templatesData}
      title="Choose a template"
      description="Using a template provides quick setup and integration of Inngest into your project. It demonstrates key functionality, allowing you to send and receive events with minimal configuration."
    />
  );
}
