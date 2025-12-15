import React from 'react';
import {
  Box,
  Flex,
  HStack,
  Step,
  StepDescription,
  StepIndicator,
  StepNumber,
  StepSeparator,
  StepStatus,
  StepTitle,
  Stepper,
  Text,
  useBreakpointValue,
  Button,
  Card,
  CardBody,
} from '@chakra-ui/react';

export interface FlowStep {
  title: string;
  description?: string;
}

interface FlowLayoutProps {
  steps: FlowStep[];
  currentStep: number;
  onBack?: () => void;
  onNext?: () => void;
  primaryLabel?: string;
  secondaryLabel?: string;
  children: React.ReactNode;
  isNextDisabled?: boolean;
}

const FlowLayout: React.FC<FlowLayoutProps> = ({
  steps,
  currentStep,
  onBack,
  onNext,
  primaryLabel,
  secondaryLabel,
  children,
  isNextDisabled,
}) => {
  const orientation = useBreakpointValue<'vertical' | 'horizontal'>({ base: 'horizontal', md: 'vertical' });

  return (
    <Flex gap={6} align="flex-start" direction={{ base: 'column', md: 'row' }}>
      <Card variant="glass" w={{ base: 'full', md: '280px' }} flexShrink={0}>
        <CardBody>
          <Stepper index={currentStep} orientation={orientation} gap={4} colorScheme="cyan">
            {steps.map((step, index) => (
              <Step key={step.title}>
                <StepIndicator>
                  <StepStatus complete={<StepNumber />} incomplete={<StepNumber />} active={<StepNumber />} />
                </StepIndicator>
                <Box flexShrink={0}>
                  <StepTitle>{step.title}</StepTitle>
                  {step.description && <StepDescription>{step.description}</StepDescription>}
                </Box>
                {index !== steps.length - 1 && <StepSeparator />}
              </Step>
            ))}
          </Stepper>
        </CardBody>
      </Card>
      <Box flex="1" w="full">
        <Card variant="surface" mb={4}>
          <CardBody>{children}</CardBody>
        </Card>
        <Card position="sticky" bottom={4} variant="glass" backdropFilter="blur(10px)" border="1px solid" borderColor="border.strong">
          <CardBody>
            <HStack justify="space-between" flexWrap="wrap" gap={3}>
              <Button variant="ghost" onClick={onBack} isDisabled={!onBack}>
                {secondaryLabel || 'Back'}
              </Button>
              <HStack spacing={3}>
                <Text color="text.muted" fontSize="sm">
                  Все шаги сохраняются автоматически
                </Text>
                {onNext && (
                  <Button onClick={onNext} isDisabled={isNextDisabled} variant="solid">
                    {primaryLabel || 'Next'}
                  </Button>
                )}
              </HStack>
            </HStack>
          </CardBody>
        </Card>
      </Box>
    </Flex>
  );
};

export default FlowLayout;
